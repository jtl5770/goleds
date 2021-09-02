package ledcontroller

import (
	"sync"
	"time"
)

type LedController struct {
	// public
	Name int
	// private
	index          int
	ledsMutex      sync.Mutex
	leds           []byte
	lastFireMutex  sync.Mutex
	lastFire       int64
	isRunningMutex sync.Mutex
	isRunning      bool
	reader         chan (*LedController)
	holdT          time.Duration
	runUpT         time.Duration
	runDownT       time.Duration
}

// public
func NewLedController(name int, size int, index int, reader chan (*LedController),
	hold time.Duration, runup time.Duration, rundown time.Duration) LedController {
	s := make([]byte, size)
	return LedController{leds: s, Name: name, index: index, isRunning: false, reader: reader,
		holdT: hold, runUpT: runup, runDownT: rundown}
}

func (s *LedController) Fire() {
	s.setLastFire()
	if s.isNotRunningAndSet() {
		go s.runner()
	}
}

func (s *LedController) GetLeds() []byte {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	ret := make([]byte, len(s.leds))
	copy(ret, s.leds)
	return ret
}

// private
func (s *LedController) setLed(index int, value byte) {
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
	s.lastFire = time.Now().UnixNano()
}

func (s *LedController) getLastFire() int64 {
	s.lastFireMutex.Lock()
	defer s.lastFireMutex.Unlock()
	return s.lastFire
}

func (s *LedController) runner() {
	left := s.index
	right := s.index

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
			s.reader <- s
			if left < 0 && right > len(s.leds)-1 {
				break
			}
			time.Sleep(s.runUpT)
		}
		// Now entering HOLD state - always after RUN_UP
		for {
			now := time.Now().UnixNano()
			hold_until := s.getLastFire() + s.holdT.Nanoseconds()
			if hold_until > now {
				time.Sleep(time.Duration(hold_until - now))
			} else {
				break
			}
		}
		// finally entering RUN DOWN state
		old_last_fire := s.getLastFire()
		for {
			last_fire := s.getLastFire()
			if last_fire > old_last_fire {
				// breaking out of inner for loop, but not outer,
				// so we are back at RUN UP with left and right preserverd
				break
			}

			if left <= s.index && left >= 0 {
				s.setLed(left, 0)
			}
			if right >= s.index && right <= len(s.leds)-1 {
				s.setLed(right, 0)
			}
			left++
			right--
			s.reader <- s
			if left > s.index && right < s.index {
				break loop
			}
			time.Sleep(s.runDownT)
		}
	}
}
