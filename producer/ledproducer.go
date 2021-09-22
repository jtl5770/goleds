package producer

import (
	"sync"
	t "time"
)

var NULL_LED = Led{0, 0, 0}

type Led struct {
	Red   byte
	Green byte
	Blue  byte
}

// True if all components are zero, false otherwise
func (s Led) IsEmpty() bool {
	return s.Red == 0 && s.Green == 0 && s.Blue == 0
}

// Return a Led with per component the max value of the caller and the
// in parameter
func (s Led) Max(in Led) Led {
	if s.Red > in.Red {
		in.Red = s.Red
	}
	if s.Green > in.Green {
		in.Green = s.Green
	}
	if s.Blue > in.Blue {
		in.Blue = s.Blue
	}
	return in
}

// The outside interface all concrete Producers need to fulfill
type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Fire()
}

// Implementation of common and shared functionality between the
// concrete Implementations
type AbstractProducer struct {
	uid       string
	leds      []Led
	isRunning bool
	lastFire  t.Time
	// Guards getting and setting LED values
	ledsMutex sync.Mutex
	// Guards changes to lastFire & isRunning
	updateMutex sync.Mutex
	ledsChanged chan (LedProducer)
	// the method Fire() should call. MUST be set by the concrete
	// implementation upon constructing a new instance
	runfunc func()
}

// Sets a single LED at index index to value
// Guarded by s.ledsMutex
func (s *AbstractProducer) setLed(index int, value Led) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	s.leds[index] = value
}

// Returns a slice with the current values of all the LEDs.
// Guarded by s.ledsMutex
func (s *AbstractProducer) GetLeds() []Led {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	ret := make([]Led, len(s.leds))
	copy(ret, s.leds)
	return ret
}

// The UID of the controller. Must be globally unique
func (s *AbstractProducer) GetUID() string {
	return s.uid
}

// Returns the time of the last time s.Fire() has been called. This is
// guarded by s.updateMutex
func (s *AbstractProducer) getLastFire() t.Time {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	return s.lastFire
}

// Used to start the main worker process as a go routine. Does never
// block.  When the worker go routine is already running, it does
// nothing besides updating s.lastFire to the current time. If the
// worker go routine is started and s.isRunning is set to true, no
// intermiediate call to Fire() will be able to start another worker
// concurrently.  The method is guarded by s.updateMutex
// IMPORTANT: After constructing your concrete instance you MUST set
// AbstractProducer.runfunc to the concrete worker method to call.
func (s *AbstractProducer) Fire() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.lastFire = t.Now()
	if !s.isRunning {
		s.isRunning = true
		go s.runfunc()
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
