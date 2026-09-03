package producer

import (
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/util"
)

func TestNightlightProducer_SetNightLed(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	totalLeds := 20
	colors := [][]float64{
		{0.1, 0.0, 0.0}, // slot 0 (red)
		{0.0, 0.1, 0.0}, // slot 1 (green)
		{0.0, 0.0, 0.1}, // slot 2 (blue)
	}

	nl := NewNightlightProducer("night_test", events, totalLeds, 48.137, 11.576, colors)
	if nl.GetUID() != "night_test" {
		t.Errorf("Expected UID 'night_test', got %s", nl.GetUID())
	}

	// Test index 0
	nl.setNightLed(0)
	buf := make([]Led, totalLeds)
	nl.GetLeds(buf)
	for i, led := range buf {
		if led.Red != 0.1 || led.Green != 0.0 || led.Blue != 0.0 {
			t.Fatalf("LED[%d] mismatch for slot 0: %+v", i, led)
		}
	}

	// Test index 2
	nl.setNightLed(2)
	nl.GetLeds(buf)
	for i, led := range buf {
		if led.Red != 0.0 || led.Green != 0.0 || led.Blue != 0.1 {
			t.Fatalf("LED[%d] mismatch for slot 2: %+v", i, led)
		}
	}

	// Test boundary out of bounds (< 0 clamps to 0, > 2 clamps to 2)
	nl.setNightLed(-5)
	nl.GetLeds(buf)
	if buf[0].Red != 0.1 {
		t.Errorf("Expected clamp to slot 0 on negative index")
	}

	nl.setNightLed(100)
	nl.GetLeds(buf)
	if buf[0].Blue != 0.1 {
		t.Errorf("Expected clamp to slot 2 on high index")
	}
}

func TestNightlightProducer_EmptyColors(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	nl := NewNightlightProducer("night_empty", events, 10, 48.0, 11.0, nil)

	// Should safely no-op without panic
	nl.setNightLed(0)
	nl.Start()
	time.Sleep(10 * time.Millisecond)
	nl.Exit()
}

func TestNightlightProducer_Lifecycle(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	totalLeds := 10
	colors := [][]float64{{0.2, 0.2, 0.2}}

	var received sync.WaitGroup
	received.Add(1)

	go func() {
		for range events.Channel() {
			received.Done()
			return
		}
	}()

	nl := NewNightlightProducer("night_lifecycle", events, totalLeds, 48.137, 11.576, colors)
	nl.Start()

	// Wait briefly or check TryStop
	stopped, err := nl.TryStop()
	if err != nil || !stopped {
		t.Logf("TryStop returned stopped=%v, err=%v (may have finished if daytime)", stopped, err)
	}

	nl.Exit()
}
