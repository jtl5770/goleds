package producer

import (
	"log/slog"
	"math"
	"time"

	"lautenbacher.net/goleds/audio"
	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

// AudioLEDProducer implements a VU meter that reads atomic audio levels
// from an AudioProvider and displays the volume on LED segments.
type AudioLEDProducer struct {
	*AbstractProducer
	provider      audio.AudioProvider
	startLedLeft  int
	endLedLeft    int
	startLedRight int
	endLedRight   int
	colors        struct {
		Green  Led
		Yellow Led
		Red    Led
	}
	updateFreq time.Duration
	minDB      float64
	maxDB      float64
}

// NewAudioLEDProducer creates a new AudioLEDProducer.
func NewAudioLEDProducer(
	uid string,
	ledsChanged *u.AtomicMapEvent[LedProducer],
	ledsTotal int,
	cfg c.AudioLEDConfig,
	provider audio.AudioProvider,
) *AudioLEDProducer {
	p := &AudioLEDProducer{
		provider:      provider,
		startLedLeft:  cfg.StartLedLeft,
		endLedLeft:    cfg.EndLedLeft,
		startLedRight: cfg.StartLedRight,
		endLedRight:   cfg.EndLedRight,
		updateFreq:    cfg.UpdateFreq,
		minDB:         cfg.MinDB,
		maxDB:         cfg.MaxDB,
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

	if p.updateFreq <= 0 {
		p.updateFreq = 30 * time.Millisecond
	}

	p.AbstractProducer = NewAbstractProducer(uid, ledsChanged, p.runner, ledsTotal)
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
	for {
		select {
		case <-p.stopchan:
			return
		case <-ticker.C:
			tickCount++
			leftDB, rightDB, active := p.provider.GetLevels()

			if tickCount%100 == 1 { // Log periodically
				slog.Debug("AudioLEDProducer polling levels", "uid", p.GetUID(), "active", active, "leftDB", leftDB, "rightDB", rightDB)
			}

			if !active || (leftDB <= p.minDB && rightDB <= p.minDB) {
				p.ClearLeds()
				continue
			}

			p.ledsMutex.Lock()
			p.updateLeds(leftDB, p.startLedLeft, p.endLedLeft)
			p.updateLeds(rightDB, p.startLedRight, p.endLedRight)
			p.ledsMutex.Unlock()

			p.ledsChanged.Send(p.GetUID(), p)
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
