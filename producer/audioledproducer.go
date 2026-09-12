package producer

import (
	"log/slog"
	"math"
	"sync/atomic"
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

type segmentRuntime struct {
	startLed int
	endLed   int
	effect   config.AudioEffectType
	barLUT   []Led
	peakLUT  []Led
}

type sceneRuntime struct {
	name     string
	segments []segmentRuntime
}

type sceneState struct {
	peaks []channelPeak
}

func brighten(c Led, factor float64) Led {
	return Led{
		Red:   min(math.Round(c.Red*factor), 255),
		Green: min(math.Round(c.Green*factor), 255),
		Blue:  min(math.Round(c.Blue*factor), 255),
	}
}

func lerpLed(a, b Led, t float64) Led {
	return Led{
		Red:   min(math.Round(a.Red + t*(b.Red-a.Red)), 255),
		Green: min(math.Round(a.Green + t*(b.Green-a.Green)), 255),
		Blue:  min(math.Round(a.Blue + t*(b.Blue-a.Blue)), 255),
	}
}

// generateSpectrumLUT precomputes a 101-entry (0% to 100%) color lookup table
// for spectrum bins linearly interpolating from low to mid (0% to 50%) and mid to high (50% to 100%).
func generateSpectrumLUT(low, mid, high Led) [101]Led {
	var lut [101]Led
	for i := 0; i <= 100; i++ {
		t := float64(i) / 100.0
		var led Led
		if t <= 0.5 {
			led = lerpLed(low, mid, t*2.0)
		} else {
			led = lerpLed(mid, high, (t-0.5)*2.0)
		}
		lut[i] = led
	}
	return lut
}

// generateGradientLUT precomputes the bar and peak LED color lookup tables
// for a segment of given length using exact transition percentages (step1, step2 in [0.0, 1.0]).
func generateGradientLUT(length int, low, mid, high Led, step1, step2 float64) (barLUT, peakLUT []Led) {
	if length <= 0 {
		return nil, nil
	}
	barLUT = make([]Led, length)
	peakLUT = make([]Led, length)

	for i := 0; i < length; i++ {
		var t float64
		if length > 1 {
			t = float64(i) / float64(length-1)
		}

		var led Led
		if t < step1 {
			led = lerpLed(low, mid, t/step1)
		} else if t < step2 {
			led = lerpLed(mid, high, (t-step1)/(step2-step1))
		} else {
			led = high
		}

		barLUT[i] = led
		peakLUT[i] = brighten(led, 1.8)
	}
	return barLUT, peakLUT
}

// AudioLEDProducer implements audio visualization (VU meter & Spectrum analyzer)
// reading atomic audio levels from an AudioProvider across configurable scenes and segments.
type AudioLEDProducer struct {
	*AbstractProducer
	provider       slimvu.AudioProvider
	scenes         []sceneRuntime
	sceneStates    []sceneState
	activeSceneIdx atomic.Int32
	vuCfg          config.AudioVUConfig
	spectrumLUT    [101]Led
	lastUpdate     time.Time
	updateFreq     time.Duration
	minDB          float64
	maxDB          float64
}

// NewAudioLEDProducer creates a new AudioLEDProducer.
func NewAudioLEDProducer(
	uid string,
	ledsChanged *util.AtomicMapEvent[LedProducer],
	ledsTotal int,
	cfg config.AudioLEDConfig,
	provider slimvu.AudioProvider,
) *AudioLEDProducer {
	vuCfg := cfg.VU
	p := &AudioLEDProducer{
		provider:   provider,
		vuCfg:      vuCfg,
		updateFreq: cfg.UpdateFreq,
		minDB:      cfg.MinDB,
		maxDB:      cfg.MaxDB,
	}

	step1 := float64(vuCfg.SwitchSteps[0]) / 100.0
	step2 := float64(vuCfg.SwitchSteps[1]) / 100.0

	colorLow := Led{Red: vuCfg.LedLow[0], Green: vuCfg.LedLow[1], Blue: vuCfg.LedLow[2]}
	colorMid := Led{Red: vuCfg.LedMid[0], Green: vuCfg.LedMid[1], Blue: vuCfg.LedMid[2]}
	colorHigh := Led{Red: vuCfg.LedHigh[0], Green: vuCfg.LedHigh[1], Blue: vuCfg.LedHigh[2]}

	specLow := Led{Red: cfg.Spectrum.LedLow[0], Green: cfg.Spectrum.LedLow[1], Blue: cfg.Spectrum.LedLow[2]}
	specMid := Led{Red: cfg.Spectrum.LedMid[0], Green: cfg.Spectrum.LedMid[1], Blue: cfg.Spectrum.LedMid[2]}
	specHigh := Led{Red: cfg.Spectrum.LedHigh[0], Green: cfg.Spectrum.LedHigh[1], Blue: cfg.Spectrum.LedHigh[2]}

	p.spectrumLUT = generateSpectrumLUT(specLow, specMid, specHigh)

	p.scenes = make([]sceneRuntime, len(cfg.Scenes))
	p.sceneStates = make([]sceneState, len(cfg.Scenes))
	for sIdx, sc := range cfg.Scenes {
		runtimeSegs := make([]segmentRuntime, len(sc.Segments))
		for segIdx, seg := range sc.Segments {
			segLen := max(seg.StartLed, seg.EndLed) - min(seg.StartLed, seg.EndLed) + 1
			barLUT, peakLUT := generateGradientLUT(segLen, colorLow, colorMid, colorHigh, step1, step2)
			runtimeSegs[segIdx] = segmentRuntime{
				startLed: seg.StartLed,
				endLed:   seg.EndLed,
				effect:   seg.Effect,
				barLUT:   barLUT,
				peakLUT:  peakLUT,
			}
		}
		p.scenes[sIdx] = sceneRuntime{
			name:     sc.Name,
			segments: runtimeSegs,
		}
		p.sceneStates[sIdx] = sceneState{
			peaks: make([]channelPeak, len(sc.Segments)),
		}
	}

	p.activeSceneIdx.Store(0)
	if cfg.ActiveScene != "" {
		for i, sc := range p.scenes {
			if sc.name == cfg.ActiveScene {
				p.activeSceneIdx.Store(int32(i))
				break
			}
		}
	}

	p.AbstractProducer = NewAbstractProducer(uid, ledsChanged, p.runner, ledsTotal)
	p.SetPriority(10)
	p.syncSpectrumEnabled()
	return p
}

// syncSpectrumEnabled activates or deactivates spectrum FFT processing on the audio provider
// depending on whether the currently active scene uses any spectrum effects.
func (p *AudioLEDProducer) syncSpectrumEnabled() {
	if p.provider == nil {
		return
	}
	activeIdx := int(p.activeSceneIdx.Load())
	hasSpectrum := false
	if activeIdx >= 0 && activeIdx < len(p.scenes) {
		for _, seg := range p.scenes[activeIdx].segments {
			if seg.effect.IsSpectrum() {
				hasSpectrum = true
				break
			}
		}
	}

	p.provider.SetSpectrumEnabled(hasSpectrum)
}

// SetActiveScene switches the active scene by name. Returns true if found and set.
func (p *AudioLEDProducer) SetActiveScene(name string) bool {
	for i, sc := range p.scenes {
		if sc.name == name {
			if int(p.activeSceneIdx.Load()) != i {
				p.activeSceneIdx.Store(int32(i))
				p.ClearLeds()
				p.syncSpectrumEnabled()
			}
			return true
		}
	}
	return false
}

// SetSceneIndex switches the active scene by index. Returns true if valid index.
func (p *AudioLEDProducer) SetSceneIndex(idx int) bool {
	if idx < 0 || idx >= len(p.scenes) {
		return false
	}
	if int(p.activeSceneIdx.Load()) != idx {
		p.activeSceneIdx.Store(int32(idx))
		p.ClearLeds()
		p.syncSpectrumEnabled()
	}
	return true
}

// NextScene cycles to the next available scene.
func (p *AudioLEDProducer) NextScene() {
	if len(p.scenes) > 1 {
		cur := p.activeSceneIdx.Load()
		next := (cur + 1) % int32(len(p.scenes))
		p.activeSceneIdx.Store(next)
		p.ClearLeds()
		p.syncSpectrumEnabled()
	}
}

// GetActiveSceneName returns the name of the currently active scene.
func (p *AudioLEDProducer) GetActiveSceneName() string {
	if len(p.scenes) == 0 {
		return ""
	}
	activeIdx := int(p.activeSceneIdx.Load())
	if activeIdx < 0 || activeIdx >= len(p.scenes) {
		return ""
	}
	return p.scenes[activeIdx].name
}

// GetSceneNames returns the names of all configured scenes.
func (p *AudioLEDProducer) GetSceneNames() []string {
	names := make([]string, len(p.scenes))
	for i, sc := range p.scenes {
		names[i] = sc.name
	}
	return names
}

// runner is the main loop polling the AudioProvider and updating LEDs.
func (p *AudioLEDProducer) runner() {
	defer func() {
		p.ClearLeds()
		if p.provider != nil {
			p.provider.SetSpectrumEnabled(false)
		}
	}()

	if p.provider == nil {
		slog.Warn("AudioLEDProducer started without AudioProvider", "uid", p.GetUID())
		return
	}

	p.syncSpectrumEnabled()

	ticker := time.NewTicker(p.updateFreq)
	defer ticker.Stop()

	var (
		spectrumLeft  [16]float32
		spectrumRight [16]float32
		spectrumMono  [16]float32
	)
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
			monoDB := (leftDB + rightDB) / 2.0

			activeIdx := int(p.activeSceneIdx.Load())
			if activeIdx < 0 || activeIdx >= len(p.scenes) {
				continue
			}
			currentScene := &p.scenes[activeIdx]
			peaks := p.sceneStates[activeIdx].peaks

			hasSpectrum := false
			for i := range currentScene.segments {
				if currentScene.segments[i].effect.IsSpectrum() {
					hasSpectrum = true
					break
				}
			}

			if hasSpectrum {
				p.provider.GetSpectrum(spectrumLeft[:], spectrumRight[:])
				for b := 0; b < 16; b++ {
					spectrumMono[b] = (spectrumLeft[b] + spectrumRight[b]) / 2.0
				}
			}

			if tickCount%100 == 1 { // Log periodically
				slog.Debug("AudioLEDProducer polling levels", "uid", p.GetUID(), "playing", playing, "leftDB", leftDB, "rightDB", rightDB)
			}

			if !playing {
				for i := range peaks {
					peaks[i] = channelPeak{}
				}

				if p.IsActive() {
					p.SetActive(false)
					p.ClearLeds()
				}
				continue
			}

			if leftDB <= p.minDB && rightDB <= p.minDB {
				hasActivePeak := false
				if p.vuCfg.PeakHoldEnabled {
					for i := range peaks {
						if peaks[i].position > 0 {
							hasActivePeak = true
							break
						}
					}
				}
				if !hasActivePeak {
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
			for i := range currentScene.segments {
				seg := &currentScene.segments[i]
				switch seg.effect {
				case config.AudioEffectLeftVU:
					p.updateVUSegment(seg, &peaks[i], leftDB, dt, now)
				case config.AudioEffectRightVU:
					p.updateVUSegment(seg, &peaks[i], rightDB, dt, now)
				case config.AudioEffectMonoVU:
					p.updateVUSegment(seg, &peaks[i], monoDB, dt, now)
				case config.AudioEffectLeftSpectrum:
					p.updateSpectrumSegment(seg, spectrumLeft[:])
				case config.AudioEffectRightSpectrum:
					p.updateSpectrumSegment(seg, spectrumRight[:])
				case config.AudioEffectMonoSpectrum:
					p.updateSpectrumSegment(seg, spectrumMono[:])
				}
			}
			p.ledsMutex.Unlock()

			p.ledsChanged.Send(p.GetUID(), p)
		}
	}
}

// updateSpectrumSegment renders the 16 frequency bands onto the segment.
func (p *AudioLEDProducer) updateSpectrumSegment(seg *segmentRuntime, bands []float32) {
	startLed := seg.startLed
	endLed := seg.endLed
	reverse := startLed > endLed
	segLen := max(startLed, endLed) - min(startLed, endLed) + 1
	if segLen < 16 {
		return
	}

	const numBins = 16
	var binWidth, gap, margin int
	if segLen >= 31 {
		gap = 1
		binWidth = (segLen - 15) / numBins
		used := (numBins * binWidth) + 15
		margin = (segLen - used) / 2
	} else {
		gap = 0
		binWidth = 1
		margin = (segLen - numBins) / 2
	}

	step := 1
	if reverse {
		step = -1
	}

	// Clear segment first (margins and gaps remain dark)
	for i := range segLen {
		p.leds[startLed+(i*step)] = Led{}
	}

	dbRange := p.maxDB - p.minDB
	for bin := 0; bin < numBins && bin < len(bands); bin++ {
		db := min(max(float64(bands[bin]), p.minDB), p.maxDB)
		var t float64
		if dbRange > 0 {
			t = (db - p.minDB) / dbRange
		}
		pct := min(max(int(math.Round(t*100.0)), 0), 100)
		binColor := p.spectrumLUT[pct]

		binStartLocal := margin + bin*(binWidth+gap)
		for w := 0; w < binWidth; w++ {
			localIdx := binStartLocal + w
			p.leds[startLed+(localIdx*step)] = binColor
		}
	}
}

// updateVUSegment sets the LED colors directly using the precomputed gradient LUTs and handles peak indicators.
func (p *AudioLEDProducer) updateVUSegment(
	seg *segmentRuntime,
	peak *channelPeak,
	db float64,
	dt float64,
	now time.Time,
) {
	startLed := seg.startLed
	endLed := seg.endLed
	reverse := startLed > endLed
	segmentLen := max(startLed, endLed) - min(startLed, endLed) + 1
	if segmentLen <= 0 || len(seg.barLUT) < segmentLen {
		return
	}

	// Clamp dB value to the expected range
	db = min(db, p.maxDB)
	db = max(db, p.minDB)

	// Normalize level from 0.0 to 1.0
	level := (db - p.minDB) / (p.maxDB - p.minDB)
	ledsToLight := int(math.Ceil(level * float64(segmentLen)))

	// Update peak tracking
	if p.vuCfg.PeakHoldEnabled {
		targetPeak := float64(ledsToLight)
		if targetPeak >= peak.position {
			peak.position = targetPeak
			peak.holdUntil = now.Add(p.vuCfg.PeakHoldTime)
			if targetPeak >= 1.0 {
				peakIdx := min(int(math.Round(targetPeak))-1, segmentLen-1)
				peak.color = seg.peakLUT[peakIdx]
			}
		} else {
			if now.After(peak.holdUntil) && dt > 0 {
				peak.position -= p.vuCfg.PeakDecayRate * dt
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

	// Single-pass direct write using directional step offset
	step := 1
	if reverse {
		step = -1
	}

	for i := range segmentLen {
		stripIndex := startLed + (i * step)
		if i < ledsToLight {
			p.leds[stripIndex] = seg.barLUT[i]
		} else {
			p.leds[stripIndex] = Led{} // Off
		}
	}

	// Draw 1-LED peak marker directly at computed physical index
	if p.vuCfg.PeakHoldEnabled && peak.position >= 1.0 {
		peakIdx := min(max(int(math.Round(peak.position))-1, 0), segmentLen-1)
		p.leds[startLed+(peakIdx*step)] = peak.color
	}
}
