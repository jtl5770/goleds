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
	uid          string
	leds         []Led
	isRunning    bool
	hasExited    bool
	ledsMutex    sync.RWMutex
	updateMutex  sync.RWMutex
	ledsChanged  *u.AtomicEvent[LedProducer]
	stopchan     chan bool
	triggerEvent *u.AtomicEvent[*u.Trigger]
	endWg        *sync.WaitGroup
	runfunc      func()
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
		endWg:        &sync.WaitGroup{},
	}
	return &inst
}

// Sets a single LED at index index to value
func (s *AbstractProducer) setLed(index int, value Led) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	s.leds[index] = value
	s.ledsChanged.Send(s)
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

// Start is the main entry point to begin the producer's execution.
func (s *AbstractProducer) Start() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	if !s.isRunning && !s.hasExited {
		s.isRunning = true
		s.endWg.Add(1)
		go s.runner()
	} else if s.hasExited || s.isRunning {
		log.Println("Start() called on AbstractProducer that is already running or has exited:", s.GetUID())
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
			log.Printf("Relaunching runner for %s due to late trigger", s.uid)
			go s.runner()
		} else {
			// No trigger was pending, it's safe to stop.
			s.isRunning = false
			s.endWg.Done()
		}
	}()

	s.runfunc()
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
