package producer

import (
	"log"
	"sync"
	t "time"

	c "lautenbacher.net/goleds/config"
)

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
	ledsChanged chan LedProducer
	// the method Fire() should call. MUST be set by the concrete
	// implementation upon constructing a new instance
	runfunc func(start t.Time)
	// this channel will be signaled via the Stop method
	stop chan bool
}

func NewAbstractProducer(uid string, ledsChanged chan LedProducer) *AbstractProducer {
	inst := AbstractProducer{
		uid:         uid,
		leds:        make([]Led, c.CONFIG.Hardware.Display.LedsTotal),
		ledsChanged: ledsChanged,
		stop:        make(chan bool, 1),
	}
	return &inst
}

// This method should only be called once per instance to make sure
// the 1 element deep buffered channel "stop" won't block
func (s *AbstractProducer) Exit() {
	s.updateMutex.Lock()
	s.runfunc = func(start t.Time) {
		log.Println("Called Start() after Exit(). Ignoring...")
	}
	s.updateMutex.Unlock()
	s.stop <- true
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

// Returns last time when s.Start() has been called. This is
// guarded by s.updateMutex
func (s *AbstractProducer) getLastStart() t.Time {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	return s.lastFire
}

// Used to start the main worker process as a go routine. Does never
// block.  When the worker go routine is already running, it does
// nothing besides updating s.lastFire to the current time. If the
// worker go routine is started and s.isRunning is set to true, no
// intermediate call to Fire() will be able to start another worker
// concurrently.  The method is guarded by s.updateMutex
// IMPORTANT: After constructing your concrete instance you MUST set
// AbstractProducer.runfunc to the concrete worker method to call.
func (s *AbstractProducer) Start() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.lastFire = t.Now()
	if !s.isRunning {
		s.isRunning = true
		go s.runfunc(s.lastFire)
	}
}

// Stop method to signal the worker go routine on the stop channel.
func (s *AbstractProducer) Stop() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.isRunning {
		// log.Println("Called Stop in " + s.GetUID())
		s.stop <- true
		// log.Println("Done calling Stop in " + s.GetUID())
	} else {
		// log.Println("Called Stop in " + s.GetUID() + " but it was not running")
	}
}

func (s *AbstractProducer) stopRunningIfNoNewFireEvent(last_fire t.Time) bool {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.lastFire.After(last_fire) {
		// again back into running up again
		return false
	} else {
		// we are finally ready and can set s.isRunning to
		// false so the next fire event can pass the mutex
		// and fire up the go routine again from the start
		s.isRunning = false
		return true
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
