package producer

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/gordonklaus/portaudio"
	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

// AudioLEDProducer implements a VU meter that reads from an audio input
// and displays the volume on a segment of LEDs.
type AudioLEDProducer struct {
	*AbstractProducer
	ledsChanged *u.AtomicMapEvent[LedProducer]
	Device      string
	startLed    int
	endLed      int
	colors      struct {
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
		ledsChanged: ledsChanged,
		startLed:    cfg.StartLed,
		endLed:      cfg.EndLed,
		Device:      cfg.Device,
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

// runner is the main processing loop for the producer.
func (p *AudioLEDProducer) runner() {
	if err := portaudio.Initialize(); err != nil {
		log.Printf("AudioLEDProducer (%s): failed to initialize portaudio: %v", p.uid, err)
		return
	}
	defer portaudio.Terminate()

	inDevice, err := p.findDevice()
	if err != nil {
		log.Printf("AudioLEDProducer (%s): %v", p.uid, err)
		return
	}

	log.Printf("AudioLEDProducer (%s): Using audio device: %s", p.uid, inDevice.Name)

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
		log.Printf("AudioLEDProducer (%s): failed to open stream: %v", p.uid, err)
		return
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		log.Printf("AudioLEDProducer (%s): failed to start stream: %v", p.uid, err)
		return
	}
	defer stream.Stop()

	ticker := time.NewTicker(p.updateFreq)
	defer ticker.Stop()

	// Clean up LEDs on exit
	defer func() {
		for i := p.startLed; i < p.endLed; i++ {
			p.leds[i] = Led{}
		}
		p.ledsChanged.Send(p.GetUID(), p)
	}()

	for {
		select {
		case <-p.stopchan:
			return
		case <-ticker.C:
			if err := stream.Read(); err != nil {
				// This can happen, e.g., portaudio.InputOverflowed. We can log it but continue.
			}

			monoSamples := stereoToMono(buffer, inDevice.MaxInputChannels)
			rms := calculateRMS(monoSamples)
			p.checkSilence(rms, ticker)
			db := rmsToDB(rms)
			p.updateLeds(db)
			p.ledsChanged.Send(p.GetUID(), p)
		}
	}
}

func (p *AudioLEDProducer) checkSilence(rms float64, ticker *time.Ticker) {
	if rms > 0 {
		if p.slowedDown {
			log.Println("AudioLEDProducer: Audio input detected, back to full loop speed...")
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
			if !p.slowedDown && time.Since(p.silenceStartTime) > 5*time.Second {
				log.Println("AudioLEDProducer: No audio input detected for 5 seconds, slowing down loop...")
				ticker.Reset(2 * time.Second)
				p.slowedDown = true
			}
		}
	}
}

// updateLeds calculates and sets the LED colors based on the dB level.
func (p *AudioLEDProducer) updateLeds(db float64) {
	segmentLen := p.endLed - p.startLed
	if segmentLen <= 0 {
		return
	}

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
		stripIndex := p.startLed + i
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
}

// findMonitorDevice attempts to find a suitable audio input device,
// preferring devices with "Monitor" in their name.
func (p *AudioLEDProducer) findDevice() (*portaudio.DeviceInfo, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("could not list audio devices: %w", err)
	}

	// Look for a squeezelite device
	for _, device := range devices {
		log.Printf("AudioLEDProducer: found device: %s", device.Name)
		if device.MaxInputChannels > 0 && strings.Contains(strings.ToLower(device.Name), p.Device) {
			return device, nil
		}
	}

	return nil, fmt.Errorf("no suitable audio input device found")
}

// stereoToMono converts a buffer of interleaved stereo samples to mono.
func stereoToMono(in []float32, channels int) []float32 {
	if channels == 1 {
		return in
	}
	numSamples := len(in) / channels
	out := make([]float32, numSamples)
	for i := range out {
		out[i] = (in[i*channels] + in[i*channels+1]) / 2.0
	}
	return out
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
	rms = max(0.001, rms) // Avoid log(0)
	return 20 * math.Log10(rms)
}
