package lc

import (
	"sync"
	"time"
)

const LEDS_TOTAL = 125
const HOLD_TIME = 5 * time.Second
const RUN_UP = 20 * time.Millisecond
const RUN_DOWN = 40 * time.Millisecond

type LedController struct {
	Name             int
	index            int
	leds_mutex       sync.Mutex
	leds             [LEDS_TOTAL]int
	last_fire_mutex  sync.Mutex
	last_fire        int64
	is_running_mutex sync.Mutex
	is_running       bool
	reader           chan (*LedController)
}

func NewLedController(name int, index int, reader chan (*LedController)) LedController {
	return LedController{Name: name, index: index, is_running: false, reader: reader}
}

func (s *LedController) setLed(index int, value int) {
	s.leds_mutex.Lock()
	defer s.leds_mutex.Unlock()
	s.leds[index] = value
}

func (s *LedController) GetLeds() [LEDS_TOTAL]int {
	s.leds_mutex.Lock()
	defer s.leds_mutex.Unlock()
	return s.leds
}

func (s *LedController) isNotRunningAndSet() bool {
	s.is_running_mutex.Lock()
	defer s.is_running_mutex.Unlock()
	if !s.is_running {
		s.is_running = true
		return true
	} else {
		return false
	}
}

func (s *LedController) unsetIsRunning() {
	s.is_running_mutex.Lock()
	defer s.is_running_mutex.Unlock()
	s.is_running = false
}

func (s *LedController) setLastFire() {
	s.last_fire_mutex.Lock()
	defer s.last_fire_mutex.Unlock()
	s.last_fire = time.Now().UnixNano()
}

func (s *LedController) getLastFire() int64 {
	s.last_fire_mutex.Lock()
	defer s.last_fire_mutex.Unlock()
	return s.last_fire
}

func (s *LedController) Fire() {
	s.setLastFire()
	if s.isNotRunningAndSet() {
		go s.runner()
	}
}

func (s *LedController) runner() {
	left := s.index
	right := s.index

	defer s.unsetIsRunning()

loop:
	for {
		for {
			if left >= 0 {
				s.setLed(left, 1)
			}
			if right <= (LEDS_TOTAL - 1) {
				s.setLed(right, 1)
			}
			right++
			left--
			s.reader <- s
			if left < 0 && right > LEDS_TOTAL-1 {
				break
			}
			time.Sleep(time.Duration(RUN_UP))
		}
		// Now entering HOLD state - always after RUN_UP
		for {
			now := time.Now().UnixNano()
			hold_until := s.getLastFire() + HOLD_TIME.Nanoseconds()
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
			if right >= s.index && right <= LEDS_TOTAL-1 {
				s.setLed(right, 0)
			}
			left++
			right--
			s.reader <- s
			if left > s.index && right < s.index {
				break loop
			}
			time.Sleep(time.Duration(RUN_DOWN))
		}
	}
}
