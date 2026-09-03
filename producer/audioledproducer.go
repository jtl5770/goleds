package producer

import (
	"log/slog"
	"math"
	"time"

	"github.com/jtl5770/go-slimvu"
	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/util"
)

type channelPeak struct {
	position  float64
	holdUntil time.Time
	color     Led
}

func brighten(c Led, factor float64) Led {
	return Led{
		Red:   min(math.Round(c.Red*factor), 255),
		Green: min(math.Round(c.Green*factor), 255),
		Blue:  min(math.Round(c.Blue*factor), 255),
	}
}

// AudioLEDProducer implements a VU meter that reads atomic audio levels
// from an AudioProvider and displays the volume on LED segments with peak hold falloff.
type AudioLEDProducer struct {
	*AbstractProducer
	provider      slimvu.AudioProvider
	startLedLeft  int
	endLedLeft    int
	startLedRight int
	endLedRight   int
	colors        struct {
		Green      Led
		Yellow     Led
		Red        Led
		PeakGreen  Led
		PeakYellow Led
		PeakRed    Led
	}
	peakHoldEnabled bool
	peakHoldTime    time.Duration
	peakDecayRate   float64 // LEDs per second
	peakLeft        channelPeak
	peakRight       channelPeak
	lastUpdate      time.Time
	updateFreq      time.Duration
	minDB           float64
	maxDB           float64
}

// NewAudioLEDProducer creates a new AudioLEDProducer.
func NewAudioLEDProducer(
	uid string,
	ledsChanged *util.AtomicMapEvent[LedProducer],
	ledsTotal int,
	cfg config.AudioLEDConfig,
	provider slimvu.AudioProvider,
) *AudioLEDProducer {
	p := &AudioLEDProducer{
		provider:        provider,
		startLedLeft:    cfg.StartLedLeft,
		endLedLeft:      cfg.EndLedLeft,
		startLedRight:   cfg.StartLedRight,
		endLedRight:     cfg.EndLedRight,
		updateFreq:      cfg.UpdateFreq,
		minDB:           cfg.MinDB,
		maxDB:           cfg.MaxDB,
		peakHoldEnabled: cfg.PeakHoldEnabled,
		peakHoldTime:    cfg.PeakHoldTime,
		peakDecayRate:   cfg.PeakDecayRate,
	}

	if len(cfg.LedGreen) >= 3 {
		p.colors.Green = Led{Red: cfg.LedGreen[0], Green: cfg.LedGreen[1], Blue: cfg.LedGreen[2]}
	}
	if len(cfg.LedYellow) >= 3 {
		p.colors.Yellow = Led{Red: cfg.LedYellow[0], Green: cfg.LedYellow[1], Blue: cfg.LedYellow[2]}
	}
	if len(cfg.LedRed) >= 3 {
		p.colors.Red = Led{Red: cfg.LedRed[0], Green: cfg.LedRed[1], Blue: cfg.LedRed[2]}
	}

	// Precompute brightened peak colors for each zone
	p.colors.PeakGreen = brighten(p.colors.Green, 1.8)
	p.colors.PeakYellow = brighten(p.colors.Yellow, 1.8)
	p.colors.PeakRed = brighten(p.colors.Red, 1.8)

	if p.peakHoldTime <= 0 {
		p.peakHoldTime = 60 * time.Millisecond
	}
	if p.peakDecayRate <= 0 {
		p.peakDecayRate = 20.0 // 20 LEDs/sec default decay rate (slower falloff)
	}

	if p.updateFreq <= 0 {
		p.updateFreq = 30 * time.Millisecond
	}

	p.AbstractProducer = NewAbstractProducer(uid, ledsChanged, p.runner, ledsTotal)
	p.SetPriority(10)
	return p
}

// runner is the main loop polling the AudioProvider and updating LEDs.
func (p *AudioLEDProducer) runner() {
	defer p.ClearLeds()

	if p.provider == nil {
		slog.Warn("AudioLEDProducer started without AudioProvider", "uid", p.GetUID())
		return
	}

	ticker := time.NewTicker(p.updateFreq)
	defer ticker.Stop()

	tickCount := 0
	p.lastUpdate = time.Now()

	for {
		select {
		case <-p.stopchan:
			return
		case <-ticker.C:
			tickCount++
			now := time.Now()
			dt := now.Sub(p.lastUpdate).Seconds()
			if dt <= 0 || dt > 1.0 {
				dt = p.updateFreq.Seconds()
			}
			p.lastUpdate = now

			leftDB, rightDB, playing := p.provider.GetLevels()

			if tickCount%100 == 1 { // Log periodically
				slog.Debug("AudioLEDProducer polling levels", "uid", p.GetUID(), "playing", playing, "leftDB", leftDB, "rightDB", rightDB)
			}

			if !playing {
				p.peakLeft = channelPeak{}
				p.peakRight = channelPeak{}
				if p.IsActive() {
					p.SetActive(false)
					p.ClearLeds()
				}
				continue
			}

			if leftDB <= p.minDB && rightDB <= p.minDB {
				if !p.peakHoldEnabled || (p.peakLeft.position <= 0 && p.peakRight.position <= 0) {
					if p.IsActive() {
						p.SetActive(false)
						p.ClearLeds()
					}
					continue
				}
			}

			if !p.IsActive() {
				p.SetActive(true)
			}

			p.ledsMutex.Lock()
			p.updateLeds(leftDB, p.startLedLeft, p.endLedLeft, &p.peakLeft, dt, now)
			p.updateLeds(rightDB, p.startLedRight, p.endLedRight, &p.peakRight, dt, now)
			p.ledsMutex.Unlock()

			p.ledsChanged.Send(p.GetUID(), p)
		}
	}
}

// updateLeds calculates and sets the LED colors based on the dB level and peak indicator.
func (p *AudioLEDProducer) updateLeds(db float64, startLed int, endLed int, peak *channelPeak, dt float64, now time.Time) {
	reverse := false
	if startLed > endLed {
		reverse = true
		startLed, endLed = endLed, startLed
	}
	segmentLen := endLed - startLed + 1
	if segmentLen <= 0 {
		return
	}

	// Clamp dB value to the expected range
	db = min(db, p.maxDB)
	db = max(db, p.minDB)

	// Normalize level from 0.0 to 1.0
	level := (db - p.minDB) / (p.maxDB - p.minDB)
	ledsToLight := int(math.Ceil(level * float64(segmentLen)))

	// Define color sections (e.g., 60% green, 25% yellow, 15% red)
	greenEnd := int(float64(segmentLen) * 0.6)
	yellowEnd := int(float64(segmentLen) * 0.85)

	// Update peak tracking
	if p.peakHoldEnabled {
		targetPeak := float64(ledsToLight)
		if targetPeak >= peak.position {
			peak.position = targetPeak
			peak.holdUntil = now.Add(p.peakHoldTime)
			if targetPeak >= 1.0 {
				peakIdx := int(math.Round(targetPeak)) - 1
				if peakIdx < greenEnd {
					peak.color = p.colors.PeakGreen
				} else if peakIdx < yellowEnd {
					peak.color = p.colors.PeakYellow
				} else {
					peak.color = p.colors.PeakRed
				}
			}
		} else {
			if now.After(peak.holdUntil) && dt > 0 {
				peak.position -= p.peakDecayRate * dt
				if peak.position < targetPeak {
					peak.position = targetPeak
				}
			}
		}
		if peak.position < 0 {
			peak.position = 0
		}
		if peak.position > float64(segmentLen) {
			peak.position = float64(segmentLen)
		}
	}

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

	// Draw 1-LED peak marker with its captured brightened zone color
	if p.peakHoldEnabled && peak.position >= 1.0 {
		peakIdx := int(math.Round(peak.position)) - 1
		if peakIdx >= segmentLen {
			peakIdx = segmentLen - 1
		}
		if peakIdx >= 0 {
			p.leds[startLed+peakIdx] = peak.color
		}
	}

	if reverse {
		for i := 0; i < segmentLen/2; i++ {
			p.leds[startLed+i], p.leds[endLed-i] = p.leds[endLed-i], p.leds[startLed+i]
		}
	}
}
