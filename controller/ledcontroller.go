package ledcontroller

import (
	"sync"
	t "time"
)

type Led byte

type LedController struct {
	// public
	UID int
	// private
	ledIndex       int
	ledsMutex      sync.Mutex
	leds           []Led
	lastFireMutex  sync.Mutex
	lastFire       t.Time
	isRunningMutex sync.Mutex
	isRunning      bool
	ledsChanged    chan (*LedController)
	holdT          t.Duration
	runUpT         t.Duration
	runDownT       t.Duration
}

// public

func NewLedController(uid int, size int, index int, ledsChanged chan (*LedController),
	hold t.Duration, runup t.Duration, rundown t.Duration) LedController {
	s := make([]Led, size)
	return LedController{leds: s, UID: uid, ledIndex: index, isRunning: false, ledsChanged: ledsChanged,
		holdT: hold, runUpT: runup, runDownT: rundown}
}

func (s *LedController) Fire() {
	s.setLastFire()
	if s.isNotRunningAndSet() {
		go s.runner()
	}
}

func (s *LedController) GetLeds() []Led {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	ret := make([]Led, len(s.leds))
	copy(ret, s.leds)
	return ret
}

// private

func (s *LedController) setLed(index int, value Led) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	s.leds[index] = value
}

func (s *LedController) isNotRunningAndSet() bool {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()
	if !s.isRunning {
		s.isRunning = true
		return true
	} else {
		return false
	}
}

func (s *LedController) unsetIsRunning() {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()
	s.isRunning = false
}

func (s *LedController) setLastFire() {
	s.lastFireMutex.Lock()
	defer s.lastFireMutex.Unlock()
	s.lastFire = t.Now()
}

func (s *LedController) getLastFire() t.Time {
	s.lastFireMutex.Lock()
	defer s.lastFireMutex.Unlock()
	return s.lastFire
}

func (s *LedController) runner() {
	left := s.ledIndex
	right := s.ledIndex

	defer s.unsetIsRunning()

loop:
	for {
		for {
			if left >= 0 {
				s.setLed(left, 255)
			}
			if right <= (len(s.leds) - 1) {
				s.setLed(right, 255)
			}
			right++
			left--
			s.ledsChanged <- s
			if left < 0 && right > len(s.leds)-1 {
				break
			}
			t.Sleep(s.runUpT)
		}
		// Now entering HOLD state - always after RUN_UP
		for {
			now := t.Now()
			hold_until := s.getLastFire().Add(s.holdT)
			if hold_until.After(now) {
				t.Sleep(t.Duration(hold_until.Sub(now)))
			} else {
				break
			}
		}
		// finally entering RUN DOWN state
		old_last_fire := s.getLastFire()
		for {
			last_fire := s.getLastFire()
			if last_fire.After(old_last_fire) {
				// breaking out of inner for loop, but not outer,
				// so we are back at RUN UP with left and right preserved
				break
			}

			if left <= s.ledIndex && left >= 0 {
				s.setLed(left, 0)
			}
			if right >= s.ledIndex && right <= len(s.leds)-1 {
				s.setLed(right, 0)
			}
			left++
			right--
			s.ledsChanged <- s
			if left > s.ledIndex && right < s.ledIndex {
				break loop
			}
			t.Sleep(s.runDownT)
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
