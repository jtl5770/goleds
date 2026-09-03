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

// generateGradientLUT precomputes the bar and peak LED color lookup tables
// for a segment of given length using anchor points centered at 0.60 and 0.85.
func generateGradientLUT(length int, green, yellow, red Led) (barLUT, peakLUT []Led) {
	if length <= 0 {
		return nil, nil
	}
	barLUT = make([]Led, length)
	peakLUT = make([]Led, length)

	for i := 0; i < length; i++ {
		var t float64
		if length > 1 {
			t = float64(i) / float64(length-1)
		} else {
			t = 0
		}

		var r, g, b float64
		// Anchor transitions centered at 0.60 and 0.85:
		// 0.00 - 0.45: Solid green
		// 0.45 - 0.75: Green -> Yellow gradient
		// 0.75 - 0.95: Yellow -> Red gradient
		// 0.95 - 1.00: Solid red
		if t < 0.45 {
			r = green.Red
			g = green.Green
			b = green.Blue
		} else if t < 0.75 {
			f := (t - 0.45) / (0.75 - 0.45)
			r = green.Red + f*(yellow.Red-green.Red)
			g = green.Green + f*(yellow.Green-green.Green)
			b = green.Blue + f*(yellow.Blue-green.Blue)
		} else if t < 0.95 {
			f := (t - 0.75) / (0.95 - 0.75)
			r = yellow.Red + f*(red.Red-yellow.Red)
			g = yellow.Green + f*(red.Green-yellow.Green)
			b = yellow.Blue + f*(red.Blue-yellow.Blue)
		} else {
			r = red.Red
			g = red.Green
			b = red.Blue
		}

		led := Led{
			Red:   min(math.Round(r), 255),
			Green: min(math.Round(g), 255),
			Blue:  min(math.Round(b), 255),
		}
		barLUT[i] = led
		peakLUT[i] = brighten(led, 1.8)
	}
	return barLUT, peakLUT
}

// AudioLEDProducer implements a VU meter that reads atomic audio levels
// from an AudioProvider and displays the volume on LED segments with peak hold falloff.
type AudioLEDProducer struct {
	*AbstractProducer
	provider        slimvu.AudioProvider
	startLedLeft    int
	endLedLeft      int
	startLedRight   int
	endLedRight     int
	leftBarLUT      []Led
	leftPeakLUT     []Led
	rightBarLUT     []Led
	rightPeakLUT    []Led
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

	var colorGreen, colorYellow, colorRed Led
	if len(cfg.LedGreen) >= 3 {
		colorGreen = Led{Red: cfg.LedGreen[0], Green: cfg.LedGreen[1], Blue: cfg.LedGreen[2]}
	}
	if len(cfg.LedYellow) >= 3 {
		colorYellow = Led{Red: cfg.LedYellow[0], Green: cfg.LedYellow[1], Blue: cfg.LedYellow[2]}
	}
	if len(cfg.LedRed) >= 3 {
		colorRed = Led{Red: cfg.LedRed[0], Green: cfg.LedRed[1], Blue: cfg.LedRed[2]}
	}

	leftLen := max(p.startLedLeft, p.endLedLeft) - min(p.startLedLeft, p.endLedLeft) + 1
	rightLen := max(p.startLedRight, p.endLedRight) - min(p.startLedRight, p.endLedRight) + 1

	p.leftBarLUT, p.leftPeakLUT = generateGradientLUT(leftLen, colorGreen, colorYellow, colorRed)
	p.rightBarLUT, p.rightPeakLUT = generateGradientLUT(rightLen, colorGreen, colorYellow, colorRed)

	if p.peakHoldTime <= 0 {
		p.peakHoldTime = 250 * time.Millisecond
	}
	if p.peakDecayRate <= 0 {
		p.peakDecayRate = 20.0 // 20 LEDs/sec default decay rate
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
			p.updateLeds(leftDB, p.startLedLeft, p.endLedLeft, &p.peakLeft, p.leftBarLUT, p.leftPeakLUT, dt, now)
			p.updateLeds(rightDB, p.startLedRight, p.endLedRight, &p.peakRight, p.rightBarLUT, p.rightPeakLUT, dt, now)
			p.ledsMutex.Unlock()

			p.ledsChanged.Send(p.GetUID(), p)
		}
	}
}

// updateLeds sets the LED colors directly using the precomputed gradient LUTs and handles peak indicators.
func (p *AudioLEDProducer) updateLeds(
	db float64,
	startLed int,
	endLed int,
	peak *channelPeak,
	barLUT []Led,
	peakLUT []Led,
	dt float64,
	now time.Time,
) {
	reverse := false
	if startLed > endLed {
		reverse = true
		startLed, endLed = endLed, startLed
	}
	segmentLen := endLed - startLed + 1
	if segmentLen <= 0 || len(barLUT) < segmentLen {
		return
	}

	// Clamp dB value to the expected range
	db = min(db, p.maxDB)
	db = max(db, p.minDB)

	// Normalize level from 0.0 to 1.0
	level := (db - p.minDB) / (p.maxDB - p.minDB)
	ledsToLight := int(math.Ceil(level * float64(segmentLen)))

	// Update peak tracking
	if p.peakHoldEnabled {
		targetPeak := float64(ledsToLight)
		if targetPeak >= peak.position {
			peak.position = targetPeak
			peak.holdUntil = now.Add(p.peakHoldTime)
			if targetPeak >= 1.0 {
				peakIdx := min(int(math.Round(targetPeak))-1, segmentLen-1)
				peak.color = peakLUT[peakIdx]
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

	// Fill the LED strip from precomputed gradient LUT
	for i := range segmentLen {
		stripIndex := startLed + i
		if i < ledsToLight {
			p.leds[stripIndex] = barLUT[i]
		} else {
			p.leds[stripIndex] = Led{} // Off
		}
	}

	// Draw 1-LED peak marker with its captured brightened gradient color
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
