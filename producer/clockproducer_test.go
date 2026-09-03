package producer

import (
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/util"
)

func TestClockProducer_NewAndSetTime(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	totalLeds := 100

	cfg := config.ClockLEDConfig{
		StartLedHour:   0,
		EndLedHour:     50,
		StartLedMinute: 51,
		EndLedMinute:   99,
		LedHour:        []float64{1.0, 0.0, 0.0},
		LedMinute:      []float64{0.0, 1.0, 0.0},
	}

	clock := NewClockProducer("clock_test", events, totalLeds, cfg)
	if clock.GetUID() != "clock_test" {
		t.Errorf("Expected UID 'clock_test', got %s", clock.GetUID())
	}

	// Directly invoke setTime to verify calculation and LED setting
	clock.setTime()

	buf := make([]Led, totalLeds)
	clock.GetLeds(buf)

	// Verify that at least one LED is set to red (hour) and one to green (minute)
	var foundHour, foundMinute bool
	for _, led := range buf {
		if led.Red == 1.0 && led.Green == 0.0 && led.Blue == 0.0 {
			foundHour = true
		}
		if led.Red == 0.0 && led.Green == 1.0 && led.Blue == 0.0 {
			foundMinute = true
		}
	}

	if !foundHour {
		t.Error("Expected hour LED to be set in buffer")
	}
	if !foundMinute {
		t.Error("Expected minute LED to be set in buffer")
	}
}

func TestClockProducer_Lifecycle(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	totalLeds := 60

	cfg := config.ClockLEDConfig{
		StartLedHour:   0,
		EndLedHour:     29,
		StartLedMinute: 30,
		EndLedMinute:   59,
		LedHour:        []float64{0.5, 0.5, 0.0},
		LedMinute:      []float64{0.0, 0.5, 0.5},
	}

	var receivedEvent sync.WaitGroup
	receivedEvent.Add(1)

	go func() {
		for range events.Channel() {
			receivedEvent.Done()
			return
		}
	}()

	clock := NewClockProducer("clock_lifecycle", events, totalLeds, cfg)
	clock.Start()

	// Wait for the first frame notification
	done := make(chan struct{})
	go func() {
		receivedEvent.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Succeeded in receiving initial clock render
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for clock producer to emit frame")
	}

	stopped, err := clock.TryStop()
	if err != nil || !stopped {
		t.Errorf("TryStop returned stopped=%v, err=%v", stopped, err)
	}

	clock.Exit()
}
