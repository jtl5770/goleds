package producer

import (
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/util"
)

func TestSensorLedProducer_RunPhases(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	totalLeds := 10
	centerIndex := 5

	cfg := config.SensorLEDConfig{
		RunUpDelay:        1 * time.Millisecond,
		RunDownDelay:      1 * time.Millisecond,
		HoldTime:          10 * time.Millisecond,
		LedRGB:            []float64{1.0, 1.0, 1.0},
		LatchEnabled:      false,
		LatchTriggerValue: 100,
		LatchTriggerDelay: 50 * time.Millisecond,
		LatchTime:         50 * time.Millisecond,
		LatchLedRGB:       []float64{1.0, 0.0, 0.0},
	}

	var endWg sync.WaitGroup
	producer := NewSensorLedProducer("sensor_test", centerIndex, events, totalLeds, cfg, &endWg)

	// Test RunUp directly
	left, right, stopped := producer.runUpPhase(centerIndex, centerIndex)
	if stopped {
		t.Error("runUpPhase unexpectedly stopped")
	}
	if left > 0 || right < totalLeds-1 {
		t.Errorf("Expected runUpPhase to cover entire strip, left=%d right=%d", left, right)
	}

	// Verify all LEDs are on
	buf := make([]Led, totalLeds)
	producer.GetLeds(buf)
	for i, led := range buf {
		if led.Red != 1.0 || led.Green != 1.0 || led.Blue != 1.0 {
			t.Errorf("LED[%d] not set after runUpPhase: %+v", i, led)
		}
	}

	// Test RunDown directly
	_, _, shouldRestart, stopped := producer.runDownPhase(0, totalLeds-1)
	if stopped || shouldRestart {
		t.Errorf("runDownPhase unexpected result: shouldRestart=%v, stopped=%v", shouldRestart, stopped)
	}

	// Verify all LEDs are off
	producer.GetLeds(buf)
	for i, led := range buf {
		if led.Red != 0.0 || led.Green != 0.0 || led.Blue != 0.0 {
			t.Errorf("LED[%d] not cleared after runDownPhase: %+v", i, led)
		}
	}
}

func TestSensorLedProducer_FullAnimationTrigger(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	totalLeds := 8
	centerIndex := 4

	cfg := config.SensorLEDConfig{
		RunUpDelay:   2 * time.Millisecond,
		RunDownDelay: 2 * time.Millisecond,
		HoldTime:     10 * time.Millisecond,
		LedRGB:       []float64{0.5, 0.5, 0.5},
		LatchLedRGB:  []float64{1.0, 0.0, 0.0},
	}

	var endWg sync.WaitGroup
	producer := NewSensorLedProducer("sensor_anim", centerIndex, events, totalLeds, cfg, &endWg)

	// Send trigger
	producer.SendTrigger(&util.Trigger{
		Value:     1,
		Timestamp: time.Now(),
	})

	// Wait for animation to finish
	done := make(chan struct{})
	go func() {
		endWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Animation finished and signaled endWg
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for SensorLedProducer animation to complete")
	}

	producer.Exit()
}

func TestSensorLedProducer_LatchMode(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	totalLeds := 6
	centerIndex := 3

	cfg := config.SensorLEDConfig{
		RunUpDelay:        1 * time.Millisecond,
		RunDownDelay:      1 * time.Millisecond,
		HoldTime:          20 * time.Millisecond,
		LedRGB:            []float64{0.2, 0.2, 0.2},
		LatchEnabled:      true,
		LatchTriggerValue: 50,
		LatchTriggerDelay: 5 * time.Millisecond,
		LatchTime:         15 * time.Millisecond,
		LatchLedRGB:       []float64{1.0, 0.0, 0.0},
	}

	producer := NewSensorLedProducer("sensor_latch", centerIndex, events, totalLeds, cfg, nil)

	// Test runLatchMode directly
	go func() {
		time.Sleep(20 * time.Millisecond)
		// Latch mode will timeout
	}()

	stopped := producer.runLatchMode()
	if stopped {
		t.Error("Expected runLatchMode to complete via timeout without being stopped")
	}
}
