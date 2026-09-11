package producer

import (
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/util"
)

type mockAudioProvider struct {
	mu              sync.Mutex
	leftDB          float64
	rightDB         float64
	spectrumLeft    [16]float32
	spectrumRight   [16]float32
	playing         bool
	spectrumEnabled bool
}

func (m *mockAudioProvider) GetLevels() (leftDB, rightDB float64, playing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.leftDB, m.rightDB, m.playing
}

func (m *mockAudioProvider) GetSpectrum(dstLeft, dstRight []float32) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	nL := copy(dstLeft, m.spectrumLeft[:])
	nR := copy(dstRight, m.spectrumRight[:])
	if nL < nR {
		return nL
	}
	return nR
}

func (m *mockAudioProvider) SetSpectrumEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spectrumEnabled = enabled
}

func (m *mockAudioProvider) IsSpectrumEnabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.spectrumEnabled
}

func (m *mockAudioProvider) SetLevels(leftDB, rightDB float64, playing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leftDB = leftDB
	m.rightDB = rightDB
	m.playing = playing
}

func (m *mockAudioProvider) SetSpectrum(leftBands, rightBands []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if leftBands != nil {
		copy(m.spectrumLeft[:], leftBands)
	}
	if rightBands != nil {
		copy(m.spectrumRight[:], rightBands)
	}
}

func (m *mockAudioProvider) Start() error { return nil }
func (m *mockAudioProvider) Stop() error  { return nil }

func TestAudioLEDProducer_LevelsUpdate(t *testing.T) {
	ledsChanged := util.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  -10.0,
		rightDB: -20.0,
		playing: true,
	}

	cfg := config.AudioLEDConfig{
		Enabled:     true,
		ActiveScene: "Stereo VU",
		Scenes: []config.AudioSceneConfig{
			{
				Name: "Stereo VU",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 9, Effect: config.AudioEffectLeftVU},
					{StartLed: 10, EndLed: 19, Effect: config.AudioEffectRightVU},
				},
			},
		},
		VU: config.AudioVUConfig{
			LedLow:          []float64{0, 100, 0},
			LedMid:          []float64{100, 100, 0},
			LedHigh:         []float64{100, 0, 0},
			SwitchSteps:     []int{60, 80},
			PeakHoldEnabled: true,
			PeakHoldTime:    250 * time.Millisecond,
			PeakDecayRate:   20.0,
		},
		Spectrum: config.AudioSpectrumConfig{
			LedLow:  []float64{0, 20, 0},
			LedMid:  []float64{50, 35, 0},
			LedHigh: []float64{140, 0, 0},
		},
		UpdateFreq: 10 * time.Millisecond,
		MinDB:      -60,
		MaxDB:      0,
	}

	p := NewAudioLEDProducer("test_audio_producer", ledsChanged, 20, cfg, mock)
	if p.GetPriority() != 10 {
		t.Errorf("Expected AudioLEDProducer priority to be 10, got %d", p.GetPriority())
	}
	p.Start()

	time.Sleep(50 * time.Millisecond)

	if !p.IsActive() {
		t.Errorf("Expected AudioLEDProducer to be active when playing audio")
	}

	buf := make([]Led, 20)
	p.GetLeds(buf)

	// Verify that at -10dB (out of -60 to 0), LEDs on the left segment (0..9) are lit
	leftLit := false
	for i := 0; i <= 9; i++ {
		if !buf[i].IsEmpty() {
			leftLit = true
			break
		}
	}
	if !leftLit {
		t.Errorf("Expected Left segment LEDs to be lit at -10dB")
	}

	// Verify that at -20dB, LEDs on the right segment (10..19) are lit
	rightLit := false
	for i := 10; i <= 19; i++ {
		if !buf[i].IsEmpty() {
			rightLit = true
			break
		}
	}
	if !rightLit {
		t.Errorf("Expected Right segment LEDs to be lit at -20dB")
	}

	// Now stop audio playback
	mock.SetLevels(-60, -60, false)
	time.Sleep(50 * time.Millisecond)

	if p.IsActive() {
		t.Errorf("Expected AudioLEDProducer to transition to isActive=false when playback stops")
	}

	p.GetLeds(buf)
	for i, led := range buf {
		if !led.IsEmpty() {
			t.Errorf("Expected LED %d to be clear after playback stops, got %v", i, led)
		}
	}

	p.Exit()
}

func TestAudioLEDProducer_SceneSwitchingAndMonoVU(t *testing.T) {
	ledsChanged := util.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  -10.0,
		rightDB: -30.0,
		playing: true,
	}

	cfg := config.AudioLEDConfig{
		Enabled:     true,
		ActiveScene: "Stereo VU",
		Scenes: []config.AudioSceneConfig{
			{
				Name: "Stereo VU",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 9, Effect: config.AudioEffectLeftVU},
					{StartLed: 10, EndLed: 19, Effect: config.AudioEffectRightVU},
				},
			},
			{
				Name: "Mono Center",
				Segments: []config.AudioSegmentConfig{
					// StartLed > EndLed animates in reverse (from 9 down to 0)
					{StartLed: 9, EndLed: 0, Effect: config.AudioEffectMonoVU},
					// StartLed <= EndLed animates forward (from 10 up to 19)
					{StartLed: 10, EndLed: 19, Effect: config.AudioEffectMonoVU},
				},
			},
			{
				Name: "Spectrum Placeholder",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 19, Effect: config.AudioEffectMonoSpectrum},
				},
			},
		},
		VU: config.AudioVUConfig{
			LedLow:          []float64{0, 100, 0},
			LedMid:          []float64{100, 100, 0},
			LedHigh:         []float64{100, 0, 0},
			SwitchSteps:     []int{60, 80},
			PeakHoldEnabled: false,
		},
		Spectrum: config.AudioSpectrumConfig{
			LedLow:  []float64{0, 20, 0},
			LedMid:  []float64{50, 35, 0},
			LedHigh: []float64{140, 0, 0},
		},
		UpdateFreq: 10 * time.Millisecond,
		MinDB:      -60,
		MaxDB:      0,
	}

	p := NewAudioLEDProducer("test_scene_switching", ledsChanged, 20, cfg, mock)
	p.Start()
	time.Sleep(30 * time.Millisecond)

	if p.GetActiveSceneName() != "Stereo VU" {
		t.Errorf("Expected active scene 'Stereo VU', got '%s'", p.GetActiveSceneName())
	}

	names := p.GetSceneNames()
	if len(names) != 3 || names[0] != "Stereo VU" || names[1] != "Mono Center" || names[2] != "Spectrum Placeholder" {
		t.Errorf("Unexpected scene names: %v", names)
	}

	// Switch to "Mono Center"
	if !p.SetActiveScene("Mono Center") {
		t.Fatalf("Failed to switch to scene 'Mono Center'")
	}
	time.Sleep(30 * time.Millisecond)
	if p.GetActiveSceneName() != "Mono Center" {
		t.Errorf("Expected active scene 'Mono Center', got '%s'", p.GetActiveSceneName())
	}

	// MonoVU level at (-10 + -30)/2 = -20dB
	buf := make([]Led, 20)
	p.GetLeds(buf)
	// Both segments 9..0 (starting from 9) and 10..19 (starting from 10) should be lit equally
	// At -20dB out of [-60, 0], level = 40/60 = 0.666 -> 7 LEDs out of 10
	// For reverse segment (9 down to 0): LEDs 9, 8, 7, 6, 5, 4, 3 are lit; 2, 1, 0 are empty
	if buf[9].IsEmpty() || buf[8].IsEmpty() || buf[3].IsEmpty() {
		t.Errorf("Expected center LEDs (e.g. 9, 8, 3) to be lit for reverse Mono segment, got buf[9]=%v, buf[3]=%v", buf[9], buf[3])
	}
	if !buf[0].IsEmpty() {
		t.Errorf("Expected LED 0 to be unlit for level 7/10 in reverse segment, got %v", buf[0])
	}

	// Test NextScene cycling
	p.NextScene()
	if p.GetActiveSceneName() != "Spectrum Placeholder" {
		t.Errorf("Expected 'Spectrum Placeholder' after NextScene, got '%s'", p.GetActiveSceneName())
	}

	p.NextScene()
	if p.GetActiveSceneName() != "Stereo VU" {
		t.Errorf("Expected 'Stereo VU' after NextScene cycle, got '%s'", p.GetActiveSceneName())
	}

	// Test SetSceneIndex
	if !p.SetSceneIndex(1) {
		t.Errorf("SetSceneIndex(1) failed")
	}
	if p.GetActiveSceneName() != "Mono Center" {
		t.Errorf("Expected active scene 'Mono Center' after SetSceneIndex(1), got '%s'", p.GetActiveSceneName())
	}

	// Invalid scene index/name
	if p.SetSceneIndex(99) {
		t.Errorf("Expected SetSceneIndex(99) to return false")
	}
	if p.SetActiveScene("InvalidName") {
		t.Errorf("Expected SetActiveScene('InvalidName') to return false")
	}

	p.Exit()
}

func TestAudioLEDProducer_StereoSpectrum(t *testing.T) {
	ledsChanged := util.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  -10.0,
		rightDB: -10.0,
		playing: true,
	}

	leftBands := make([]float32, 16)
	rightBands := make([]float32, 16)
	for i := 0; i < 16; i++ {
		leftBands[i] = -60.0
		rightBands[i] = -60.0
	}
	// Left band 0 is full power (0 dB)
	leftBands[0] = 0.0
	// Right band 15 is full power (0 dB)
	rightBands[15] = 0.0
	mock.SetSpectrum(leftBands, rightBands)

	specLow := Led{Red: 0, Green: 20, Blue: 0}
	specMid := Led{Red: 50, Green: 35, Blue: 0}
	specHigh := Led{Red: 140, Green: 0, Blue: 0}

	cfg := config.AudioLEDConfig{
		Enabled:     true,
		ActiveScene: "Stereo Spectrum",
		Scenes: []config.AudioSceneConfig{
			{
				Name: "Stereo Spectrum",
				Segments: []config.AudioSegmentConfig{
					// Segment 1: LeftSpectrum (20 LEDs, 0..19). margin=2, bins at 2..17
					{StartLed: 0, EndLed: 19, Effect: config.AudioEffectLeftSpectrum},
					// Segment 2: RightSpectrum (20 LEDs, 20..39). margin=2, bins at 22..37
					{StartLed: 20, EndLed: 39, Effect: config.AudioEffectRightSpectrum},
					// Segment 3: MonoSpectrum (20 LEDs, 40..59). margin=2, bins at 42..57
					{StartLed: 40, EndLed: 59, Effect: config.AudioEffectMonoSpectrum},
				},
			},
		},
		VU: config.AudioVUConfig{
			LedLow:          []float64{0, 80, 0},
			LedMid:          []float64{40, 40, 0},
			LedHigh:         []float64{100, 0, 0},
			SwitchSteps:     []int{60, 80},
			PeakHoldEnabled: false,
		},
		Spectrum: config.AudioSpectrumConfig{
			LedLow:  []float64{specLow.Red, specLow.Green, specLow.Blue},
			LedMid:  []float64{specMid.Red, specMid.Green, specMid.Blue},
			LedHigh: []float64{specHigh.Red, specHigh.Green, specHigh.Blue},
		},
		UpdateFreq: 10 * time.Millisecond,
		MinDB:      -60,
		MaxDB:      0,
	}

	p := NewAudioLEDProducer("test_stereo_spectrum", ledsChanged, 60, cfg, mock)
	p.Start()
	time.Sleep(50 * time.Millisecond)

	buf := make([]Led, 60)
	p.GetLeds(buf)

	// Left spectrum: bin 0 (LED 2) should be specHigh (0 dB)
	if buf[2] != specHigh {
		t.Errorf("Expected LeftSpectrum bin 0 (LED 2) to be specHigh %v, got %v", specHigh, buf[2])
	}
	// Left spectrum: bin 15 (LED 17) should be specLow (-60 dB)
	if buf[17] != specLow {
		t.Errorf("Expected LeftSpectrum bin 15 (LED 17) to be specLow %v, got %v", specLow, buf[17])
	}

	// Right spectrum: bin 0 (LED 22) should be specLow (-60 dB)
	if buf[22] != specLow {
		t.Errorf("Expected RightSpectrum bin 0 (LED 22) to be specLow %v, got %v", specLow, buf[22])
	}
	// Right spectrum: bin 15 (LED 37) should be specHigh (0 dB)
	if buf[37] != specHigh {
		t.Errorf("Expected RightSpectrum bin 15 (LED 37) to be specHigh %v, got %v", specHigh, buf[37])
	}

	// Mono spectrum: average of (0 + -60)/2 = -30 dB (50% energy -> specMid)
	// bin 0 (LED 42) and bin 15 (LED 57) should both be around specMid
	if buf[42] != specMid {
		t.Errorf("Expected MonoSpectrum bin 0 (LED 42) to be specMid %v, got %v", specMid, buf[42])
	}
	if buf[57] != specMid {
		t.Errorf("Expected MonoSpectrum bin 15 (LED 57) to be specMid %v, got %v", specMid, buf[57])
	}

	p.Exit()
}

func TestAudioLEDProducer_SpectrumCPUGating(t *testing.T) {
	ledsChanged := util.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  -10.0,
		rightDB: -10.0,
		playing: true,
	}

	cfg := config.AudioLEDConfig{
		Enabled:     true,
		ActiveScene: "VU Only",
		Scenes: []config.AudioSceneConfig{
			{
				Name: "VU Only",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 9, Effect: config.AudioEffectLeftVU},
				},
			},
			{
				Name: "Spectrum Scene",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 19, Effect: config.AudioEffectLeftSpectrum},
				},
			},
		},
		VU: config.AudioVUConfig{
			LedLow:          []float64{0, 80, 0},
			LedMid:          []float64{40, 40, 0},
			LedHigh:         []float64{100, 0, 0},
			SwitchSteps:     []int{60, 80},
			PeakHoldEnabled: false,
		},
		Spectrum: config.AudioSpectrumConfig{
			LedLow:  []float64{0, 20, 0},
			LedMid:  []float64{50, 35, 0},
			LedHigh: []float64{140, 0, 0},
		},
		UpdateFreq: 10 * time.Millisecond,
		MinDB:      -60,
		MaxDB:      0,
	}

	p := NewAudioLEDProducer("test_cpu_gating", ledsChanged, 20, cfg, mock)

	// Initially active scene is "VU Only" -> spectrum processing should be disabled
	if mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrumEnabled=false when active scene has no spectrum segments")
	}

	// Switch to "Spectrum Scene" -> spectrum processing should be enabled
	p.SetActiveScene("Spectrum Scene")
	if !mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrumEnabled=true when active scene has spectrum segments")
	}

	// Switch back to "VU Only" -> spectrum processing should be disabled
	p.SetActiveScene("VU Only")
	if mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrumEnabled=false after switching back to VU scene")
	}

	// NextScene should switch to Spectrum Scene
	p.NextScene()
	if !mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrumEnabled=true after NextScene switched to Spectrum Scene")
	}

	// Start producer and then exit -> ensure spectrum is disabled on exit
	p.Start()
	time.Sleep(30 * time.Millisecond)
	p.Exit()
	time.Sleep(30 * time.Millisecond)

	if mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrumEnabled=false after AudioLEDProducer exit")
	}
}

func TestAudioLEDProducer_DynamicPeakHoldAndDecay(t *testing.T) {
	ledsChanged := util.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  0.0,   // Max volume -> Red zone (index 9)
		rightDB: -40.0, // Low volume -> Green zone
		playing: true,
	}

	cfg := config.AudioLEDConfig{
		Enabled:     true,
		ActiveScene: "Stereo",
		Scenes: []config.AudioSceneConfig{
			{
				Name: "Stereo",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 9, Effect: config.AudioEffectLeftVU},
					{StartLed: 10, EndLed: 19, Effect: config.AudioEffectRightVU},
				},
			},
		},
		VU: config.AudioVUConfig{
			LedLow:          []float64{0, 80, 0},
			LedMid:          []float64{40, 40, 0},
			LedHigh:         []float64{100, 0, 0},
			SwitchSteps:     []int{60, 80},
			PeakHoldEnabled: true,
			PeakHoldTime:    80 * time.Millisecond,
			PeakDecayRate:   10.0, // 10 LEDs per second
		},
		Spectrum: config.AudioSpectrumConfig{
			LedLow:  []float64{0, 20, 0},
			LedMid:  []float64{50, 35, 0},
			LedHigh: []float64{140, 0, 0},
		},
		UpdateFreq: 10 * time.Millisecond,
		MinDB:      -60,
		MaxDB:      0,
	}

	p := NewAudioLEDProducer("test_peak_producer", ledsChanged, 20, cfg, mock)
	p.Start()

	time.Sleep(30 * time.Millisecond)

	buf := make([]Led, 20)
	p.GetLeds(buf)
	// Left channel at max (0 dB) -> LED 9 must be lit
	if buf[9].IsEmpty() {
		t.Fatalf("Expected LED 9 to be lit on max level, but was empty")
	}
	expectedPeakHigh := buf[9]

	// Drop left level down to -60 dB
	mock.SetLevels(-60.0, -40.0, true)

	// During the hold window (< 80ms), LED 9 should STILL be lit in PeakHigh
	time.Sleep(30 * time.Millisecond)
	p.GetLeds(buf)
	if buf[9] != expectedPeakHigh {
		t.Errorf("Expected LED 9 to remain held in PeakHigh %v, got %v", expectedPeakHigh, buf[9])
	}

	// Wait past hold time and let it decay down
	time.Sleep(150 * time.Millisecond)
	p.GetLeds(buf)
	// Peak should have decayed down from index 9, while retaining its captured PeakHigh color
	peakFound := false
	for i := 0; i <= 9; i++ {
		if buf[i] == expectedPeakHigh {
			peakFound = true
			if i >= 9 {
				t.Errorf("Expected peak to have decayed below index 9, but found at %d", i)
			}
			break
		}
	}
	if !peakFound {
		t.Errorf("Expected PeakHigh marker to still be visible while decaying")
	}

	p.Exit()
}

func TestGenerateGradientLUT(t *testing.T) {
	low := Led{Red: 0, Green: 255, Blue: 0}
	mid := Led{Red: 255, Green: 255, Blue: 0}
	high := Led{Red: 255, Green: 0, Blue: 0}

	// Test 0 length
	b0, p0 := generateGradientLUT(0, low, mid, high, 0.60, 0.80)
	if len(b0) != 0 || len(p0) != 0 {
		t.Errorf("Expected empty LUTs for length 0")
	}

	// Test 1 length
	b1, p1 := generateGradientLUT(1, low, mid, high, 0.60, 0.80)
	if len(b1) != 1 || len(p1) != 1 {
		t.Fatalf("Expected LUT length 1, got %d", len(b1))
	}
	if b1[0] != low {
		t.Errorf("Expected single LED to be low %v, got %v", low, b1[0])
	}

	// Test 20 LEDs
	b20, p20 := generateGradientLUT(20, low, mid, high, 0.60, 0.80)
	if len(b20) != 20 || len(p20) != 20 {
		t.Fatalf("Expected LUT length 20, got %d", len(b20))
	}

	// Index 0 should be pure low
	if b20[0] != low {
		t.Errorf("Expected index 0 to be low, got %v", b20[0])
	}

	// Index 19 should be pure high
	if b20[19] != high {
		t.Errorf("Expected index 19 to be high, got %v", b20[19])
	}

	// Mid index (around 60% = index 12) should have both high and low components
	if b20[12].Red == 0 || b20[12].Green == 0 {
		t.Errorf("Expected index 12 to be transitional gradient, got %v", b20[12])
	}

	// Peak LUT tests: check brightening and capping at 255
	if p20[0].Green != 255 {
		t.Errorf("Expected peakLUT[0].Green capped at 255, got %f", p20[0].Green)
	}
	if p20[19].Red != 255 {
		t.Errorf("Expected peakLUT[19].Red capped at 255, got %f", p20[19].Red)
	}
}

func BenchmarkAudioLEDProducer_UpdateVUSegment(b *testing.B) {
	low := Led{Red: 0, Green: 255, Blue: 0}
	mid := Led{Red: 255, Green: 255, Blue: 0}
	high := Led{Red: 255, Green: 0, Blue: 0}
	barLUT, peakLUT := generateGradientLUT(30, low, mid, high, 0.60, 0.80)

	p := &AudioLEDProducer{
		AbstractProducer: &AbstractProducer{
			leds: make([]Led, 30),
		},
		minDB: -60,
		maxDB: 0,
		vuCfg: config.AudioVUConfig{
			PeakHoldEnabled: true,
			PeakHoldTime:    250 * time.Millisecond,
			PeakDecayRate:   20.0,
		},
	}

	seg := segmentRuntime{
		startLed: 0,
		endLed:   29,
		effect:   config.AudioEffectLeftVU,
		barLUT:   barLUT,
		peakLUT:  peakLUT,
	}

	now := time.Now()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.updateVUSegment(&seg, -12.5, 0.030, now)
	}
}

func TestAudioLEDProducer_UpdateSpectrumSegment(t *testing.T) {
	specLow := Led{Red: 0, Green: 20, Blue: 0}
	specMid := Led{Red: 50, Green: 35, Blue: 0}
	specHigh := Led{Red: 140, Green: 0, Blue: 0}
	lut := generateSpectrumLUT(specLow, specMid, specHigh)

	// Test LUT endpoints and exact linear midpoint
	if lut[0] != specLow {
		t.Errorf("Expected lut[0] to be specLow %v, got %v", specLow, lut[0])
	}
	if lut[50] != specMid {
		t.Errorf("Expected lut[50] to be exact midpoint specMid %v, got %v", specMid, lut[50])
	}
	if lut[100] != specHigh {
		t.Errorf("Expected lut[100] to be specHigh %v, got %v", specHigh, lut[100])
	}

	p := &AudioLEDProducer{
		AbstractProducer: &AbstractProducer{
			leds: make([]Led, 60),
		},
		minDB:       -60,
		maxDB:       0,
		spectrumLUT: lut,
	}

	t.Run("Segment >= 31 LEDs has 1-LED gaps and equal bin width", func(t *testing.T) {
		// segLen = 47: 15 gaps, 16 bins of width 2 = 32 LEDs, total used = 47, margin = 0
		seg := segmentRuntime{
			startLed: 0,
			endLed:   46,
			effect:   config.AudioEffectMonoSpectrum,
		}
		bands := make([]float32, 16)
		bands[0] = 0.0   // Max energy -> should be specHigh
		bands[1] = -60.0 // Min energy -> should be empty Led{}
		bands[2] = 0.0   // Max energy -> should be specHigh

		p.updateSpectrumSegment(&seg, bands)

		// Bin 0: LEDs 0, 1
		if p.leds[0] != specHigh || p.leds[1] != specHigh {
			t.Errorf("Expected Bin 0 (LEDs 0, 1) to be specHigh, got %v, %v", p.leds[0], p.leds[1])
		}
		// Gap 0: LED 2
		if !p.leds[2].IsEmpty() {
			t.Errorf("Expected Gap at LED 2 to be empty, got %v", p.leds[2])
		}
		// Bin 1: LEDs 3, 4 (minDB -> specLow)
		if p.leds[3] != specLow || p.leds[4] != specLow {
			t.Errorf("Expected Bin 1 (LEDs 3, 4) to be specLow at minDB, got %v, %v", p.leds[3], p.leds[4])
		}
		// Gap 1: LED 5
		if !p.leds[5].IsEmpty() {
			t.Errorf("Expected Gap at LED 5 to be empty, got %v", p.leds[5])
		}
		// Bin 2: LEDs 6, 7 (0dB -> specHigh)
		if p.leds[6] != specHigh || p.leds[7] != specHigh {
			t.Errorf("Expected Bin 2 (LEDs 6, 7) to be specHigh, got %v, %v", p.leds[6], p.leds[7])
		}
	})

	t.Run("Segment < 31 LEDs has 0 gaps and is centered", func(t *testing.T) {
		// segLen = 20: 16 bins of width 1, margin = (20-16)/2 = 2 LEDs (LEDs 0, 1 and 18, 19 are dark)
		seg := segmentRuntime{
			startLed: 0,
			endLed:   19,
			effect:   config.AudioEffectMonoSpectrum,
		}
		bands := make([]float32, 16)
		bands[0] = 0.0 // Max energy -> specHigh

		p.updateSpectrumSegment(&seg, bands)

		// Margins
		if !p.leds[0].IsEmpty() || !p.leds[1].IsEmpty() {
			t.Errorf("Expected left margins (LEDs 0, 1) to be empty, got %v, %v", p.leds[0], p.leds[1])
		}
		// Bin 0 at margin offset 2
		if p.leds[2] != specHigh {
			t.Errorf("Expected Bin 0 at LED 2 to be specHigh, got %v", p.leds[2])
		}
	})
}

func BenchmarkAudioLEDProducer_UpdateSpectrumSegment(b *testing.B) {
	lut := generateSpectrumLUT(Led{Red: 0, Green: 20, Blue: 0}, Led{Red: 50, Green: 35, Blue: 0}, Led{Red: 140, Green: 0, Blue: 0})
	p := &AudioLEDProducer{
		AbstractProducer: &AbstractProducer{leds: make([]Led, 60)},
		minDB:            -60,
		maxDB:            0,
		spectrumLUT:      lut,
	}
	seg := segmentRuntime{startLed: 0, endLed: 46, effect: config.AudioEffectMonoSpectrum}
	bands := []float32{-10, -20, -5, -30, -15, -40, -8, -12, -25, -35, -50, -18, -22, -6, -14, -28}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p.updateSpectrumSegment(&seg, bands)
	}
}

func TestAudioLEDProducer_SpectrumFFTGating(t *testing.T) {
	ledsChanged := util.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  -10.0,
		rightDB: -20.0,
		playing: true,
	}

	cfg := config.AudioLEDConfig{
		Enabled:     true,
		ActiveScene: "VU Only",
		Scenes: []config.AudioSceneConfig{
			{
				Name: "VU Only",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 9, Effect: config.AudioEffectLeftVU},
					{StartLed: 10, EndLed: 19, Effect: config.AudioEffectRightVU},
				},
			},
			{
				Name: "With Spectrum",
				Segments: []config.AudioSegmentConfig{
					{StartLed: 0, EndLed: 19, Effect: config.AudioEffectMonoSpectrum},
				},
			},
		},
		VU: config.AudioVUConfig{
			LedLow:          []float64{0, 255, 0},
			LedMid:          []float64{255, 255, 0},
			LedHigh:         []float64{255, 0, 0},
			SwitchSteps:     []int{60, 80},
			PeakHoldEnabled: true,
			PeakHoldTime:    250 * time.Millisecond,
			PeakDecayRate:   15.0,
		},
		Spectrum: config.AudioSpectrumConfig{
			LedLow:  []float64{0, 20, 0},
			LedMid:  []float64{50, 35, 0},
			LedHigh: []float64{140, 0, 0},
		},
		UpdateFreq: 30 * time.Millisecond,
		MinDB:      -60,
		MaxDB:      0,
	}

	p := NewAudioLEDProducer("test_audio", ledsChanged, 20, cfg, mock)
	defer p.Exit()

	// 1. "VU Only" scene is active -> spectrum FFT must be shut off
	if mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrum FFT to be disabled for VU-only scene, but was enabled")
	}

	// 2. Switch to scene containing MonoSpectrum -> spectrum FFT must be enabled
	if !p.SetActiveScene("With Spectrum") {
		t.Fatalf("Failed to set active scene to 'With Spectrum'")
	}
	if !mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrum FFT to be enabled for scene with MonoSpectrum, but was disabled")
	}

	// 3. Switch back to "VU Only" -> spectrum FFT must be shut off again
	if !p.SetActiveScene("VU Only") {
		t.Fatalf("Failed to set active scene to 'VU Only'")
	}
	if mock.IsSpectrumEnabled() {
		t.Errorf("Expected spectrum FFT to be disabled after switching back to VU-only scene, but was enabled")
	}
}
