// This is the main producer: reacting to a sensor trigger to light
// the stripes starting at the position of the sensor and moving
// outwards in both directions. The producer is configured with a hold
// time (how long the stripes should stay fully lit after the sensor
// has triggered) and a run-up and run-down time (how long it takes to
// light up the whole stripe and how long it takes to turn off the
// whole stripe). The producer reacts to new sensor triggers while it
// is running by extending the hold time and by switching back to
// run-up if it is already in run-down. The producer is configured
// with one color for the LEDs on the stripes. The producer is stopped
// when the hold time has expired and there have been no new sensor
// triggers in the meantime. The producer switches back to run-up mode
// when a new sensor trigger is received while it is running in
// run-down mode.

package producer

import (
	"time"
	t "time"

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
func (s *SensorLedProducer) runner(starttime t.Time) {
	defer func() {
		s.setIsRunning(false)
	}()

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
				// log.Println("Stopped SensorLedProducer...")
				ticker.Stop()
				return
			}
		}
		// Now entering HOLD state - always, uconditionally after
		// RUN_UP is complete. If there have been any Fire() events in
		// the meantime or if there are more during hold, the hold
		// period will be extended to be at least the last Fire()
		// event time plus s.holdT
		var old_last_start time.Time
		for {
			now := time.Now()
			last_start := s.getLastStart()
			hold_until := last_start.Add(s.holdT)
			if hold_until.After(now) {
				select {
				case <-time.After(hold_until.Sub(now)):
					// continue
				case <-s.stop:
					// log.Println("Stopped SensorLedProducer...")
					ticker.Stop()
					return
				}
			} else {
				// make sure to store the last looked at Fire() event
				// time so we don't accidentally loose events. If
				// there have been new ones, we will see in the
				// RUN_DOWN section and skip back to the beginning
				old_last_start = last_start
				break
			}
		}
		// finally entering RUN DOWN state
		ticker.Reset(s.runDownT)
		for {
			last_start := s.getLastStart()
			if last_start.After(old_last_start) {
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

				if !s.getLastStart().After(last_start) {
					// we are finally ready and can end the go routine
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
				// log.Println("Stopped SensorLedProducer...")
				return
			}
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
