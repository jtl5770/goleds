package producer

import (
	"errors"
	"log/slog"
	"sync"
	t "time"

	u "lautenbacher.net/goleds/util"
)

var errTimeout = errors.New("timeout reached while sending stop signal")

// Implementation of common and shared functionality between the
// concrete Implementations of the ledproducer interface
type AbstractProducer struct {
	uid          string
	leds         []Led
	isRunning    bool
	hasExited    bool
	ledsMutex    sync.RWMutex
	updateMutex  sync.RWMutex
	ledsChanged  *u.AtomicMapEvent[LedProducer]
	stopchan     chan bool
	triggerEvent *u.AtomicEvent[*u.Trigger]
	endWg        *sync.WaitGroup
	runfunc      func()
}

// Creates a new instance of AbstractProducer. The uid must be unique
func NewAbstractProducer(uid string, ledsChanged *u.AtomicMapEvent[LedProducer], runfunc func(), ledsTotal int) *AbstractProducer {
	inst := AbstractProducer{
		uid:          uid,
		leds:         make([]Led, ledsTotal),
		ledsChanged:  ledsChanged,
		stopchan:     make(chan bool),
		runfunc:      runfunc,
		triggerEvent: u.NewAtomicEvent[*u.Trigger](),
		endWg:        &sync.WaitGroup{},
	}
	return &inst
}

// Sets a single LED at index index to value
func (s *AbstractProducer) setLed(index int, value Led) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	s.leds[index] = value
	s.ledsChanged.Send(s.GetUID(), s)
}

// Returns a slice with the current values of all the LEDs.
func (s *AbstractProducer) GetLeds() []Led {
	s.ledsMutex.RLock()
	defer s.ledsMutex.RUnlock()
	ret := make([]Led, len(s.leds))
	copy(ret, s.leds)
	return ret
}

// The UID of the controller. Must be globally unique
func (s *AbstractProducer) GetUID() string {
	return s.uid
}

// startLocked is the internal, non-locking version of Start.
// It MUST be called with updateMutex held.
func (s *AbstractProducer) startLocked() {
	s.isRunning = true
	s.endWg.Add(1)
	go s.runner()
}

// Start is the main entry point to begin the producer's execution.
func (s *AbstractProducer) Start() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if !s.isRunning && !s.hasExited {
		s.startLocked()
	} else if s.hasExited {
		slog.Warn("Start() called on a Producer that has already exited", "uid", s.GetUID())
	} else if s.isRunning {
		slog.Warn("Start() called on a Producer that is already running", "uid", s.GetUID())
	}
}

// runner is the central goroutine for a producer. It calls the concrete
// implementation's runfunc and includes logic to handle the race condition
// where a trigger arrives just as the animation is finishing.
func (s *AbstractProducer) runner() {
	defer func() {
		s.updateMutex.Lock()
		defer s.updateMutex.Unlock()

		// This is the core of the race condition fix.
		// After the runfunc completes, we do a final non-destructive check for a trigger.
		if s.triggerEvent.HasPending() {
			// A trigger was pending. Relaunch the runner to handle it.
			// The pending notification remains in the channel for the new runner to consume.
			slog.Info("Relaunching runner due to late trigger", "uid", s.uid)
			go s.runner()
		} else {
			// No trigger was pending, it's safe to stop.
			s.isRunning = false
			s.endWg.Done()
		}
	}()

	s.runfunc()
}

// SendTrigger ensures the producer is running and then sends it a trigger event.
// This operation is atomic, preventing a race between starting and triggering.
func (s *AbstractProducer) SendTrigger(trigger *u.Trigger) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if !s.hasExited {
		// Ensure the producer is running before sending the trigger.
		if !s.isRunning {
			s.startLocked()
		}
		s.triggerEvent.Send(trigger)
	}
}

// TryStop attempts to signal the worker goroutine to stop.
// It returns (true, nil) if the signal was sent successfully.
// It returns (false, nil) if the producer was not running.
// It returns (false, err) if a timeout occurs.
func (s *AbstractProducer) TryStop() (bool, error) {
	s.updateMutex.RLock()
	defer s.updateMutex.RUnlock()

	if !s.isRunning || s.hasExited {
		slog.Debug("TryStop called on a producer that was not running", "uid", s.GetUID())
		return false, nil
	}

	select {
	case s.stopchan <- true:
		return true, nil
	case <-t.After(5 * t.Second):
		slog.Warn("Timeout reached while sending stop signal", "uid", s.GetUID())
		return false, errTimeout
	}
}

// This method should only be called once per instance
func (s *AbstractProducer) Exit() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	close(s.stopchan)
	s.isRunning = false
	s.hasExited = true
}
