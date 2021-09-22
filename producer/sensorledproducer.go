package producer

import (
	"time"
)

type SensorLedProducer struct {
	AbstractProducer
	ledIndex int
	holdT    time.Duration
	runUpT   time.Duration
	runDownT time.Duration
	ledOn    Led
}

func NewSensorLedProducer(uid string, size int, index int, ledsChanged chan (LedProducer),
	hold time.Duration, runup time.Duration, rundown time.Duration, ledOn Led) *SensorLedProducer {
	leds := make([]Led, size)
	inst := &SensorLedProducer{
		AbstractProducer: AbstractProducer{
			leds:        leds,
			uid:         uid,
			isRunning:   false,
			ledsChanged: ledsChanged,
		},
		ledIndex: index,
		holdT:    hold,
		runUpT:   runup,
		runDownT: rundown,
		ledOn:    ledOn}
	inst.runfunc = inst.runner
	return inst
}

// Sets a single LED at index index to value
// Guarded by s.ledsMutex
func (s *SensorLedProducer) setLed(index int, value Led) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	s.leds[index] = value
}

// The main worker, doing a run-up, hold, and run-down cycle (if
// undisturbed by intermediate Fire() events). It checks for these
// intermediate Fire() events during hold time (to prolong the hold
// time accordingly) and during run-down to switch back into the
// run-up part if needed. At the end it checks one last time for an
// intermediate Fire() before finally setting s.isRunning to false and
// ending the go routine. All this is either quarded directly or
// indirectly (by calls to s.getLastFire()) by s.updateMutex.
func (s *SensorLedProducer) runner() {
	left := s.ledIndex
	right := s.ledIndex

loop:
	for {
		ticker := time.NewTicker(s.runUpT)
		for {
			if left >= 0 {
				s.setLed(left, s.ledOn)
			}
			if right <= len(s.leds)-1 {
				s.setLed(right, s.ledOn)
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
		var old_last_fire time.Time
		for {
			now := time.Now()
			last_fire := s.getLastFire()
			hold_until := last_fire.Add(s.holdT)
			if hold_until.After(now) {
				time.Sleep(time.Duration(hold_until.Sub(now)))
			} else {
				// make sure to store the last looked at Fire() event
				// time so we don't accidentally loose events. If
				// there have been new ones, we will see in the
				// RUN_DOWN section and skip back to the beginning
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
				s.setLed(left, NULL_LED)
			}
			if right >= s.ledIndex && right <= len(s.leds)-1 {
				s.setLed(right, NULL_LED)
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
