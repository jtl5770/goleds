package producer

import (
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/util"
)

type mockAudioProvider struct {
	mu      sync.Mutex
	leftDB  float64
	rightDB float64
	playing bool
}

func (m *mockAudioProvider) GetLevels() (leftDB, rightDB float64, playing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.leftDB, m.rightDB, m.playing
}

func (m *mockAudioProvider) SetLevels(leftDB, rightDB float64, playing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leftDB = leftDB
	m.rightDB = rightDB
	m.playing = playing
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
		StartLedLeft:  0,
		EndLedLeft:    9,
		StartLedRight: 10,
		EndLedRight:   19,
		UpdateFreq:    10 * time.Millisecond,
		MinDB:         -60,
		MaxDB:         0,
		LedGreen:      []float64{0, 100, 0},
		LedYellow:     []float64{100, 100, 0},
		LedRed:        []float64{100, 0, 0},
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

func TestAudioLEDProducer_DynamicPeakHoldAndDecay(t *testing.T) {
	ledsChanged := util.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  0.0,  // Max volume -> Red zone (index 9)
		rightDB: -40.0, // Low volume -> Green zone
		playing: true,
	}

	cfg := config.AudioLEDConfig{
		StartLedLeft:    0,
		EndLedLeft:      9,
		StartLedRight:   10,
		EndLedRight:     19,
		UpdateFreq:      10 * time.Millisecond,
		MinDB:           -60,
		MaxDB:           0,
		LedGreen:        []float64{0, 80, 0},
		LedYellow:       []float64{40, 40, 0},
		LedRed:          []float64{100, 0, 0},
		PeakHoldEnabled: true,
		PeakHoldTime:    80 * time.Millisecond,
		PeakDecayRate:   10.0, // 10 LEDs per second
	}

	p := NewAudioLEDProducer("test_peak_producer", ledsChanged, 20, cfg, mock)
	p.Start()

	time.Sleep(30 * time.Millisecond)

	buf := make([]Led, 20)
	p.GetLeds(buf)

	// Highest LED for left channel (index 9) must be PeakRed (brightened Red)
	expectedPeakRed := brighten(Led{Red: 100, Green: 0, Blue: 0}, 1.8)
	if buf[9] != expectedPeakRed {
		t.Errorf("Expected LED 9 to be PeakRed %v, got %v", expectedPeakRed, buf[9])
	}

	// Now drop the left audio level to silence (-60dB)
	mock.SetLevels(-60.0, -60.0, true)

	// During the hold window (< 80ms), LED 9 should STILL be lit in PeakRed
	time.Sleep(30 * time.Millisecond)
	p.GetLeds(buf)
	if buf[9] != expectedPeakRed {
		t.Errorf("Expected LED 9 to remain held in PeakRed %v, got %v", expectedPeakRed, buf[9])
	}

	// Wait past hold time and let it decay down
	time.Sleep(150 * time.Millisecond)
	p.GetLeds(buf)
	// Peak should have decayed down from index 9, while retaining its captured PeakRed color
	peakFound := false
	for i := 0; i <= 9; i++ {
		if buf[i] == expectedPeakRed {
			peakFound = true
			if i >= 9 {
				t.Errorf("Expected peak to have decayed below index 9, but found at %d", i)
			}
			break
		}
	}
	if !peakFound {
		t.Errorf("Expected PeakRed marker to still be visible while decaying")
	}

	p.Exit()
}

func TestGenerateGradientLUT(t *testing.T) {
	green := Led{Red: 0, Green: 255, Blue: 0}
	yellow := Led{Red: 255, Green: 255, Blue: 0}
	red := Led{Red: 255, Green: 0, Blue: 0}

	// Test 0 length
	b0, p0 := generateGradientLUT(0, green, yellow, red)
	if len(b0) != 0 || len(p0) != 0 {
		t.Errorf("Expected empty LUTs for length 0")
	}

	// Test 1 length
	b1, p1 := generateGradientLUT(1, green, yellow, red)
	if len(b1) != 1 || len(p1) != 1 {
		t.Fatalf("Expected LUT length 1, got %d", len(b1))
	}
	if b1[0] != green {
		t.Errorf("Expected single LED to be green %v, got %v", green, b1[0])
	}

	// Test 20 LEDs
	b20, p20 := generateGradientLUT(20, green, yellow, red)
	if len(b20) != 20 || len(p20) != 20 {
		t.Fatalf("Expected LUT length 20, got %d", len(b20))
	}

	// Index 0 should be pure green
	if b20[0] != green {
		t.Errorf("Expected index 0 to be green, got %v", b20[0])
	}

	// Index 19 should be pure red
	if b20[19] != red {
		t.Errorf("Expected index 19 to be red, got %v", b20[19])
	}

	// Mid index (around 60% = index 12) should have both red and green
	if b20[12].Red == 0 || b20[12].Green == 0 {
		t.Errorf("Expected index 12 to be transitional gradient, got %v", b20[12])
	}

	// Peak LUT should be brightened version of bar LUT
	for i := range b20 {
		expectedPeak := brighten(b20[i], 1.8)
		if p20[i] != expectedPeak {
			t.Errorf("Peak LUT mismatch at index %d: expected %v, got %v", i, expectedPeak, p20[i])
		}
	}
}

func BenchmarkAudioLEDProducer_UpdateLeds(b *testing.B) {
	green := Led{Red: 0, Green: 255, Blue: 0}
	yellow := Led{Red: 255, Green: 255, Blue: 0}
	red := Led{Red: 255, Green: 0, Blue: 0}
	barLUT, peakLUT := generateGradientLUT(30, green, yellow, red)

	p := &AudioLEDProducer{
		AbstractProducer: &AbstractProducer{
			leds: make([]Led, 30),
		},
		minDB:           -60,
		maxDB:           0,
		peakHoldEnabled: true,
		peakHoldTime:    250 * time.Millisecond,
		peakDecayRate:   20.0,
	}

	now := time.Now()
	peak := channelPeak{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.updateLeds(-12.5, 0, 29, &peak, barLUT, peakLUT, 0.030, now)
	}
}
