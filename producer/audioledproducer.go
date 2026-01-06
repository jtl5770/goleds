// +build cgo

package producer

import (
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

var (
	paMutex       sync.Mutex
	paInitialized bool
)

// AudioLEDProducer implements a VU meter that reads from an audio input
// and displays the volume on a segment of LEDs.
type AudioLEDProducer struct {
	*AbstractProducer
	ledsChanged   *u.AtomicMapEvent[LedProducer]
	Device        string
	startLedLeft  int
	endLedLeft    int
	startLedRight int
	endLedRight   int
	colors        struct {
		Green  Led
		Yellow Led
		Red    Led
	}
	sampleRate       int
	framesPerBuffer  int
	updateFreq       time.Duration
	minDB            float64
	maxDB            float64
	silenceStartTime time.Time
	silenceStart     bool
	slowedDown       bool
}

// NewAudioLEDProducer creates a new AudioLEDProducer.
func NewAudioLEDProducer(uid string, ledsChanged *u.AtomicMapEvent[LedProducer], ledsTotal int, cfg c.AudioLEDConfig) *AudioLEDProducer {
	p := &AudioLEDProducer{
		ledsChanged:   ledsChanged,
		startLedLeft:  cfg.StartLedLeft,
		endLedLeft:    cfg.EndLedLeft,
		startLedRight: cfg.StartLedRight,
		endLedRight:   cfg.EndLedRight,
		Device:        cfg.Device,
	}
	p.colors.Green = Led{Red: cfg.LedGreen[0], Green: cfg.LedGreen[1], Blue: cfg.LedGreen[2]}
	p.colors.Yellow = Led{Red: cfg.LedYellow[0], Green: cfg.LedYellow[1], Blue: cfg.LedYellow[2]}
	p.colors.Red = Led{Red: cfg.LedRed[0], Green: cfg.LedRed[1], Blue: cfg.LedRed[2]}
	p.sampleRate = cfg.SampleRate
	p.framesPerBuffer = cfg.FramesPerBuffer
	p.updateFreq = cfg.UpdateFreq
	p.minDB = cfg.MinDB
	p.maxDB = cfg.MaxDB
	p.silenceStart = false
	p.slowedDown = false
	p.AbstractProducer = NewAbstractProducer(uid, ledsChanged, p.runner, ledsTotal)
	return p
}

func (p *AudioLEDProducer) Exit() {
	p.AbstractProducer.Exit()
	paMutex.Lock()
	defer paMutex.Unlock()
	if paInitialized {
		if err := portaudio.Terminate(); err != nil {
			slog.Error("AudioLEDProducer: failed to terminate portaudio", "uid", p.uid, "error", err)
		} else {
			slog.Info("AudioLEDProducer: PortAudio terminated.")
			paInitialized = false
		}
	}
}

// runner is the main processing loop for the producer.
func (p *AudioLEDProducer) runner() {
	paMutex.Lock()
	if !paInitialized {
		if err := portaudio.Initialize(); err != nil {
			slog.Error("AudioLEDProducer: failed to initialize portaudio", "uid", p.uid, "error", err)
			paMutex.Unlock()
			return
		}
		slog.Info("AudioLEDProducer: PortAudio initialized.")
	}
	paInitialized = true
	paMutex.Unlock()

	inDevice, err := p.findDevice()
	if err != nil {
		slog.Error("AudioLEDProducer: no device", "uid", p.GetUID(), "error", err)
		return
	}

	slog.Info("AudioLEDProducer", "uid", p.GetUID(), "device", inDevice.Name, "sampleRate", p.sampleRate, "framesPerBuffer", p.framesPerBuffer)

	buffer := make([]float32, p.framesPerBuffer*inDevice.MaxInputChannels)
	streamParams := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   inDevice,
			Channels: inDevice.MaxInputChannels,
			Latency:  inDevice.DefaultLowInputLatency,
		},
		SampleRate:      float64(p.sampleRate),
		FramesPerBuffer: p.framesPerBuffer,
	}

	stream, err := portaudio.OpenStream(streamParams, buffer)
	if err != nil {
		slog.Error("AudioLEDProducer: failed to open stream", "uid", p.uid, "error", err)
		return
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		slog.Error("AudioLEDProducer: failed to start stream", "uid", p.uid, "error", err)
		return
	}
	defer stream.Stop()

	ticker := time.NewTicker(p.updateFreq)
	defer ticker.Stop()

	// Clean up LEDs on exit
	defer func() {
		p.leds = make([]Led, len(p.leds))
		p.ledsChanged.Send(p.GetUID(), p)
	}()

	p.slowedDown = false
	p.silenceStart = false
	for {
		select {
		case <-p.stopchan:
			return
		case <-ticker.C:
			if p.slowedDown {
				stream, err = portaudio.OpenStream(streamParams, buffer)
				if err != nil {
					slog.Error("AudioLEDProducer: failed to open stream", "uid", p.uid, "error", err)
					return
				}
				if err = stream.Start(); err != nil {
					slog.Error("AudioLEDProducer: failed to start stream", "uid", p.uid, "error", err)
					return
				}
			}
			if err = stream.Read(); err != nil {
				// This can happen, e.g., portaudio.InputOverflowed. We can log it but continue.
			}

			samplesL, samplesR := deInterleave(buffer, inDevice.MaxInputChannels)
			rmsL := calculateRMS(samplesL)
			rmsR := calculateRMS(samplesR)
			p.checkSilence(rmsL, rmsR, ticker)
			if p.slowedDown {
				stream.Stop()
				stream.Close()
			}

			dbL := rmsToDB(rmsL)
			dbR := rmsToDB(rmsR)
			p.updateLeds(dbL, p.startLedLeft, p.endLedLeft)
			p.updateLeds(dbR, p.startLedRight, p.endLedRight)
			p.ledsChanged.Send(p.GetUID(), p)
		}
	}
}

func (p *AudioLEDProducer) checkSilence(rmsL float64, rmsR float64, ticker *time.Ticker) {
	if rmsL > 0 || rmsR > 0 {
		if p.slowedDown {
			slog.Info("AudioLEDProducer: Audio input detected, back to full loop speed...")
			p.silenceStart = false
			p.slowedDown = false
			ticker.Reset(p.updateFreq)
		} else if p.silenceStart {
			// Reset silence start if we detect audio after a period of silence
			p.silenceStart = false
		}
	} else {
		if !p.silenceStart {
			p.silenceStart = true
			p.silenceStartTime = time.Now()
		} else {
			if !p.slowedDown && time.Since(p.silenceStartTime) > 10*time.Second {
				slog.Info("AudioLEDProducer: No audio input detected for 10 seconds, slowing down loop...")
				ticker.Reset(5 * time.Second)
				p.slowedDown = true
			}
		}
	}
}

// updateLeds calculates and sets the LED colors based on the dB level.
func (p *AudioLEDProducer) updateLeds(db float64, startLed int, endLed int) {
	reverse := false
	if startLed > endLed {
		reverse = true
		startLed, endLed = endLed, startLed
	}
	segmentLen := endLed - startLed + 1

	// Clamp dB value to the expected range
	db = min(db, p.maxDB)
	db = max(db, p.minDB)

	// Normalize level from 0.0 to 1.0
	level := (db - p.minDB) / (p.maxDB - p.minDB)
	ledsToLight := int(math.Ceil(level * float64(segmentLen)))

	// Define color sections (e.g., 70% green, 20% yellow, 10% red)
	greenEnd := int(float64(segmentLen) * 0.7)
	yellowEnd := int(float64(segmentLen) * 0.9)

	for i := range segmentLen {
		stripIndex := startLed + i
		if i < ledsToLight {
			if i < greenEnd {
				p.leds[stripIndex] = p.colors.Green
			} else if i < yellowEnd {
				p.leds[stripIndex] = p.colors.Yellow
			} else {
				p.leds[stripIndex] = p.colors.Red
			}
		} else {
			p.leds[stripIndex] = Led{} // Off
		}
	}
	if reverse {
		for i := 0; i < segmentLen/2; i++ {
			p.leds[startLed+i], p.leds[endLed-i] = p.leds[endLed-i], p.leds[startLed+i]
		}
	}
}

func (p *AudioLEDProducer) findDevice() (*portaudio.DeviceInfo, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("could not list audio devices: %w", err)
	}

	// Look for a squeezelite device
	for _, device := range devices {
		// log.Printf("AudioLEDProducer: found device: %s", device.Name)
		if device.MaxInputChannels > 0 && strings.Contains(strings.ToLower(device.Name), p.Device) {
			return device, nil
		}
	}

	return nil, fmt.Errorf("no suitable audio input device found")
}

// deInterleave converts a buffer of interleaved stereo samples to mono.
func deInterleave(in []float32, channels int) ([]float32, []float32) {
	if channels == 1 {
		return in, in
	}
	numSamples := len(in) / channels
	outL := make([]float32, numSamples)
	outR := make([]float32, numSamples)
	for i := range numSamples {
		outL[i] = in[channels*i]
		outR[i] = in[channels*i+1]
	}
	return outL, outR
}

// calculateRMS calculates the Root Mean Square of a slice of audio samples.
func calculateRMS(samples []float32) float64 {
	var sumSquare float64
	for _, sample := range samples {
		sumSquare += float64(sample * sample)
	}
	meanSquare := sumSquare / float64(len(samples))
	return math.Sqrt(meanSquare)
}

// rmsToDB converts an RMS value (0.0-1.0) to a decibel scale.
func rmsToDB(rms float64) float64 {
	rms = max(0.0001, rms) // Avoid log(0)
	return 20 * math.Log10(rms)
}
