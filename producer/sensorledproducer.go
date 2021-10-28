package producer

import (
	"log"
	"time"

	c "lautenbacher.net/goleds/config"
)

type SensorLedProducer struct {
	*AbstractProducer
	ledIndex int
	holdT    time.Duration
	runUpT   time.Duration
	runDownT time.Duration
	ledOn    Led
}

func NewSensorLedProducer(uid string, index int, ledsChanged chan (LedProducer)) *SensorLedProducer {
	inst := SensorLedProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
		ledIndex:         index,
		holdT:            c.CONFIG.SensorLED.HoldTime,
		runUpT:           c.CONFIG.SensorLED.RunUpDelay,
		runDownT:         c.CONFIG.SensorLED.RunDownDelay,
		ledOn: Led{
			Red:   c.CONFIG.SensorLED.LedRGB[0],
			Green: c.CONFIG.SensorLED.LedRGB[1],
			Blue:  c.CONFIG.SensorLED.LedRGB[2],
		},
	}
	inst.runfunc = inst.runner
	return &inst
}

// The main worker, doing a run-up, hold, and run-down cycle (if
// undisturbed by intermediate Fire() events). It checks for these
// intermediate Fire() events during hold time (to prolong the hold
// time accordingly) and during run-down to switch back into the
// run-up part if needed. At the end it checks one last time for an
// intermediate Fire() before finally setting s.isRunning to false and
// ending the go routine. All this is either guarded directly or
// indirectly (by calls to s.getLastFire()) by s.updateMutex.
func (s *SensorLedProducer) runner() {
	left := s.ledIndex
	right := s.ledIndex

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
			select {
			case <-ticker.C:
			// continue
			case <-s.stop:
				ticker.Stop()
				log.Println("Stopped SensorLedProducer...")
				return
			}
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
				select {
				case <-time.After(hold_until.Sub(now)):
					// continue
				case <-s.stop:
					log.Println("Stopped SensorLedProducer...")
					return
				}
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
				s.setLed(left, Led{})
			}
			if right >= s.ledIndex && right <= len(s.leds)-1 {
				s.setLed(right, Led{})
			}
			s.ledsChanged <- s
			if left == s.ledIndex && right == s.ledIndex {
				// that means: we have run down completely. Now we
				// either simply end the go routine (allowing for a
				// fire event to trigger a new complete run up, hold,
				// run down cycle in the future or - as a last check -
				// we see if there has been a fire event in the little
				// time while this last iteration of the inner for
				// loop took place thereby closing a small race condition)
				ticker.Stop()
				if s.stopRunningIfNoNewFire(last_fire) {
					// we are finally ready and can return and end the
					// go routine
					return
				} else {
					// back into running up again
					break
				}
			}
			left++
			right--
			select {
			case <-ticker.C:
				// continue
			case <-s.stop:
				ticker.Stop()
				log.Println("Stopped SensorLedProducer...")
				return
			}
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
