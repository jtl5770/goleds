package producer

import (
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/util"
)

func TestAbstractProducer_PriorityAndActive(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	ap := NewAbstractProducer("test_ap", events, func() {}, 10)

	if ap.GetPriority() != 0 {
		t.Errorf("Expected default priority 0, got %d", ap.GetPriority())
	}
	ap.SetPriority(42)
	if ap.GetPriority() != 42 {
		t.Errorf("Expected priority 42, got %d", ap.GetPriority())
	}

	if !ap.IsActive() {
		t.Error("Expected default isActive to be true")
	}
	ap.SetActive(false)
	if ap.IsActive() {
		t.Error("Expected isActive to be false")
	}
}

func TestAbstractProducer_ClearAndGetLeds(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	ap := NewAbstractProducer("test_leds", events, func() {}, 5)

	ap.ledsMutex.Lock()
	ap.leds[0] = Led{Red: 1.0, Green: 0.5, Blue: 0.25}
	ap.ledsMutex.Unlock()

	buf := make([]Led, 5)
	ap.GetLeds(buf)
	if buf[0].Red != 1.0 || buf[0].Green != 0.5 || buf[0].Blue != 0.25 {
		t.Errorf("Unexpected LED values: %+v", buf[0])
	}

	ap.ClearLeds()
	ap.GetLeds(buf)
	if buf[0].Red != 0.0 || buf[0].Green != 0.0 || buf[0].Blue != 0.0 {
		t.Errorf("Expected cleared LEDs, got: %+v", buf[0])
	}
}

func TestAbstractProducer_Lifecycle(t *testing.T) {
	events := util.NewAtomicMapEvent[LedProducer]()
	var started sync.WaitGroup
	started.Add(1)

	var ap *AbstractProducer
	ap = NewAbstractProducer("test_life", events, func() {
		started.Done()
		select {
		case <-ap.stopchan:
			return
		case <-time.After(1 * time.Second):
			return
		}
	}, 10)

	ap.Start()
	started.Wait()

	// Calling Start again on running producer should be safe no-op
	ap.Start()

	stopped, err := ap.TryStop()
	if err != nil {
		t.Errorf("TryStop failed: %v", err)
	}
	if !stopped {
		t.Error("Expected TryStop to return true")
	}

	ap.Exit()
	// Calling Start on exited producer should be safe no-op
	ap.Start()
}
