package producer

import (
	"testing"
	"time"

	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

type mockAudioProvider struct {
	leftDB  float64
	rightDB float64
	active  bool
}

func (m *mockAudioProvider) GetLevels() (leftDB, rightDB float64, active bool) {
	return m.leftDB, m.rightDB, m.active
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
		LedGreen:      []float64{0, 255, 0},
		LedYellow:     []float64{255, 255, 0},
		LedRed:        []float64{255, 0, 0},
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
