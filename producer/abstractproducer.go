package producer

import (
	"log"
	"sync"
	t "time"

	c "lautenbacher.net/goleds/config"
)

// Implementation of common and shared functionality between the
// concrete Implementations of the ledproducer interface
type AbstractProducer struct {
	uid       string
	leds      []Led
	isRunning bool
	hasExited bool
	lastStart t.Time
	// Guards getting and setting LED values
	ledsMutex sync.RWMutex
	// Guards changes to lastStart & isRunning & hasExited
	updateMutex sync.RWMutex
	ledsChanged chan LedProducer
	// the method Start() should call. MUST be set by the concrete
	// implementation upon constructing a new instance
	runfunc func(start t.Time)
	// this channel will be signaled via the Stop method. Your runfunc
	// MUST listen to this channel and exit when it receives a signal
	stop chan bool
}

// Creates a new instance of AbstractProducer. The uid must be unique
func NewAbstractProducer(uid string, ledsChanged chan LedProducer) *AbstractProducer {
	inst := AbstractProducer{
		uid:         uid,
		leds:        make([]Led, c.CONFIG.Hardware.Display.LedsTotal),
		ledsChanged: ledsChanged,
		stop:        make(chan bool),
	}
	return &inst
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

// Returns last time when s.Start() has been called. This is
// guarded by s.updateMutex
func (s *AbstractProducer) getLastStart() t.Time {
	s.updateMutex.RLock()
	defer s.updateMutex.RUnlock()

	return s.lastStart
}

// Used to start the main worker process as a go routine. Does never
// block.  When the worker go routine is already running, it does
// nothing besides updating s.lastStart to the current time. If the
// worker go routine is started and s.isRunning is set to true, no
// intermediate call to Fire() will be able to start another worker
// concurrently.  The method is guarded by s.updateMutex
// IMPORTANT: After constructing your concrete instance you MUST set
// AbstractProducer.runfunc to the concrete worker method to call.
func (s *AbstractProducer) Start() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.lastStart = t.Now()
	if !s.isRunning && !s.hasExited {
		s.isRunning = true
		go s.runfunc(s.lastStart)
	}
}

// Stop method to signal the worker go routine on the stop channel.
func (s *AbstractProducer) Stop() {
	s.updateMutex.RLock()
	defer s.updateMutex.RUnlock()
	if s.isRunning && !s.hasExited {
		go func() {
			select {
			case s.stop <- true:
			case <-t.After(1 * t.Second):
				log.Println("Timeout in ", s.GetUID(), ": could NOT send stop signal")
			}
		}()
	}
}

// This method should only be called once per instance
func (s *AbstractProducer) Exit() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.isRunning {
		close(s.stop)
	}
	s.hasExited = true
}

func (s *AbstractProducer) GetIsRunning() bool {
	s.updateMutex.RLock()
	defer s.updateMutex.RUnlock()
	return s.isRunning
}

func (s *AbstractProducer) setIsRunning(running bool) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	s.isRunning = running
}
