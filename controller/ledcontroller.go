package ledcontroller

import (
	"sync"
	t "time"
)

type Led byte

type LedController struct {
	UID       int
	ledIndex  int
	leds      []Led
	isRunning bool
	lastFire  t.Time
	holdT     t.Duration
	runUpT    t.Duration
	runDownT  t.Duration
	// Guards getting and setting LED values
	ledsMutex sync.Mutex
	// Guards changes to lastFire & isRunning
	updateMutex sync.Mutex
	ledsChanged chan (*LedController)
}

func NewLedController(uid int, size int, index int, ledsChanged chan (*LedController),
	hold t.Duration, runup t.Duration, rundown t.Duration) *LedController {
	s := make([]Led, size)
	return &LedController{leds: s, UID: uid, ledIndex: index, isRunning: false, ledsChanged: ledsChanged,
		holdT: hold, runUpT: runup, runDownT: rundown}
}

// Returns a slice with the current values of all the LEDs.
// Guarded by s.ledsMutex
func (s *LedController) GetLeds() []Led {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	ret := make([]Led, len(s.leds))
	copy(ret, s.leds)
	return ret
}

// Sets a single LED at index index to value
// Guarded by s.ledsMutex
func (s *LedController) setLed(index int, value Led) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	s.leds[index] = value
}

// This public method can be called whenever the associated sensor
// reads a value above its trigger point. When the s.runner go routine
// is already running, it does nothing besides updating s.lastFire to
// the current time. If the s.runner go routine is started and
// s.isRunning is set to true, so no intermiediate call to Fire() will
// be able to start another runner concurrently.
// The method is guarded by s.updateMutex
func (s *LedController) Fire() {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	s.lastFire = t.Now()
	if !s.isRunning {
		s.isRunning = true
		go s.runner()
	}
}

// Return the s.lastFire value, guarded by s.updateMutex
func (s *LedController) getLastFire() t.Time {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	return s.lastFire
}

// The main worker, doing a run-up, hold, and run-down cycle (if
// undisturbed by intermediate Fire() events). It checks for these
// intermediate Fire() events during hold time (to prolong the hold
// time accordingly) and during run-down to switch back into the
// run-up part if needed. At the end it checks one last time for an
// intermediate Fire() before finally setting s.isRunning to false and
// ending the go routine. All this is either quarded directly or
// indirectly (by calls to s.getLastFire()) by s.updateMutex.
func (s *LedController) runner() {
	left := s.ledIndex
	right := s.ledIndex

loop:
	for {
		ticker := t.NewTicker(s.runUpT)
		for {
			if left >= 0 {
				s.setLed(left, 255)
			}
			if right <= len(s.leds)-1 {
				s.setLed(right, 255)
			}
			s.ledsChanged <- s
			if left <= 0 && right >= len(s.leds)-1 {
				ticker.Stop()
				break
			}
			right++
			left--
			<-ticker.C
		}
		// Now entering HOLD state - always, uconditionally after
		// RUN_UP is complete. If there have been any Fire() events in
		// the meantime or if there are more during hold, the hold
		// period will be extended to be at least the last Fire()
		// event time plus s.holdT
		var old_last_fire t.Time
		for {
			now := t.Now()
			last_fire := s.getLastFire()
			hold_until := last_fire.Add(s.holdT)
			if hold_until.After(now) {
				t.Sleep(t.Duration(hold_until.Sub(now)))
			} else {
				// make sure to store the last looked at Fire() event
				// time so we don'taccidentally loose events. If there
				// have been new ones, we will see in the RUN_DOWN section
				// and skip back to the beginning
				old_last_fire = last_fire
				break
			}
		}
		// finally entering RUN DOWN state
		ticker.Reset(s.runDownT)
		for {
			last_fire := s.getLastFire()
			if last_fire.After(old_last_fire) {
				// breaking out of inner for loop, but not outer, so
				// we are back at RUN UP while preserving the current
				// value for left and right
				ticker.Stop()
				break
			}

			if left <= s.ledIndex && left >= 0 {
				s.setLed(left, 0)
			}
			if right >= s.ledIndex && right <= len(s.leds)-1 {
				s.setLed(right, 0)
			}
			s.ledsChanged <- s
			if left == s.ledIndex && right == s.ledIndex {
				// that means: we have run down completely. Now we
				// either simply end the go routine (allowing for a
				// fire event to trigger a new complete run up, hold,
				// run down cycle in the future or - as a last check -
				// we see if there has been a fire event in the little
				// time while this last iteration of the inner for
				// loop took place)
				s.updateMutex.Lock()
				ticker.Stop()
				if s.lastFire.After(last_fire) {
					s.updateMutex.Unlock()
					// again back into running up again
					break
				} else {
					// we are finally ready and can set s.isRunning to
					// false so the next fire event can pass the mutex
					// and fire up the go routine again from the start
					s.isRunning = false
					s.updateMutex.Unlock()
					break loop
				}
			}
			left++
			right--
			<-ticker.C
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
