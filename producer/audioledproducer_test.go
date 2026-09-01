package producer

import (
	"sync"
	"testing"
	"time"

	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

type mockAudioProvider struct {
	mu      sync.Mutex
	leftDB  float64
	rightDB float64
	active  bool
}

func (m *mockAudioProvider) GetLevels() (leftDB, rightDB float64, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.leftDB, m.rightDB, m.active
}

func (m *mockAudioProvider) SetLevels(leftDB, rightDB float64, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leftDB = leftDB
	m.rightDB = rightDB
	m.active = active
}

func (m *mockAudioProvider) Start() error { return nil }
func (m *mockAudioProvider) Stop() error  { return nil }

func TestAudioLEDProducer_LevelsUpdate(t *testing.T) {
	ledsChanged := u.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  -10.0,
		rightDB: -20.0,
		active:  true,
	}

	cfg := c.AudioLEDConfig{
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
	p.Start()

	time.Sleep(50 * time.Millisecond)

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

	p.Exit()
}

func TestAudioLEDProducer_DynamicPeakHoldAndDecay(t *testing.T) {
	ledsChanged := u.NewAtomicMapEvent[LedProducer]()
	mock := &mockAudioProvider{
		leftDB:  0.0, // Max volume -> Red zone (index 9)
		rightDB: -40.0, // Low volume -> Green zone
		active:  true,
	}

	cfg := c.AudioLEDConfig{
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
