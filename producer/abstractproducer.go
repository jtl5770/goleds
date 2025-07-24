package producer

import (
	"log"
	"sync"
	t "time"

	u "lautenbacher.net/goleds/util"
)

// Implementation of common and shared functionality between the
// concrete Implementations of the ledproducer interface
type AbstractProducer struct {
	uid       string
	leds      []Led
	isRunning bool
	hasExited bool
	// Guards getting and setting LED values
	ledsMutex sync.RWMutex
	// Guards changes to lastStart & isRunning & hasExited
	updateMutex sync.RWMutex
	ledsChanged *u.AtomicEvent[LedProducer]
	// this channel will be signaled via the Stop method. Your runfunc
	// MUST listen to this channel and exit when it receives a signal
	stopchan chan bool
	// this AtomicEvent contains the channel that is written to by the
	// SendTrigger(*u.Trigger) method whenever it is called. Your
	// runfunc MAY read from this channel if it is interested in those
	// triggers. Only the newest Trigger will be available.
	triggerEvent *u.AtomicEvent[*u.Trigger]
	// the method Start() should call. It is set via NewAbstractProducer.
	runfunc func()
}

// Creates a new instance of AbstractProducer. The uid must be unique
func NewAbstractProducer(uid string, ledsChanged *u.AtomicEvent[LedProducer], runfunc func(), ledsTotal int) *AbstractProducer {
	inst := AbstractProducer{
		uid:          uid,
		leds:         make([]Led, ledsTotal),
		ledsChanged:  ledsChanged,
		stopchan:     make(chan bool),
		runfunc:      runfunc,
		triggerEvent: u.NewAtomicEvent[*u.Trigger](),
	}
	return &inst
}

// Sets a single LED at index index to value
// Guarded by s.ledsMutex
func (s *AbstractProducer) setLed(index int, value Led) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	s.leds[index] = value
	s.ledsChanged.Send(s)
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

// Used to start the main worker process as a go routine. Does never
// block.  When the worker go routine is already running, it does
// nothing besides updating s.lastTrigger to the current trigger. If the
// worker go routine is started and s.isRunning is set to true, no
// intermediate call to Start() will be able to start another worker
// concurrently.  The method is guarded by s.updateMutex
// IMPORTANT: After constructing your concrete instance you MUST set
// AbstractProducer.runfunc to the concrete worker method to call.
func (s *AbstractProducer) Start() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if !s.isRunning && !s.hasExited {
		s.isRunning = true
		go s.runfunc()
	} else if s.hasExited || s.isRunning {
		log.Println("Start() called on AbstractProducer that is already running or has exited:", s.GetUID())
	}
}

func (s *AbstractProducer) SendTrigger(trigger *u.Trigger) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if s.isRunning && !s.hasExited {
		s.triggerEvent.Send(trigger)
	}
}

// Stop method to signal the worker go routine on the stop channel.
func (s *AbstractProducer) Stop() {
	s.updateMutex.RLock()
	defer s.updateMutex.RUnlock()
	if s.isRunning && !s.hasExited {
		select {
		case s.stopchan <- true:
		case <-t.After(5 * t.Second):
			log.Println("Timeout reached in ", s.GetUID(),
				": blocked sending stop signal")
		}
	}
}

// This method should only be called once per instance
func (s *AbstractProducer) Exit() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	if s.isRunning {
		close(s.stopchan)
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
