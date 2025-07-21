// SensorLedProducer creates a light animation on an LED strip when a sensor is
// triggered. The animation is a "pulse" of light originating from the sensor's
// position, expanding to fill the entire strip, holding for a set duration,
// and then contracting back to the center before turning off.
//
// It is designed to be responsive, meaning its behavior changes if new sensor
// triggers arrive while an animation is already in progress.
//
// # Key Responsibilities & Behavior
//
// 1. Configuration:
//   - ledIndex: The starting position of the animation, corresponding to the
//     sensor's physical location.
//   - ledOn: The color of the illuminated LEDs.
//   - runUpT: The delay between steps as the light expands outwards.
//   - runDownT: The delay between steps as the light contracts inwards.
//   - holdT: The minimum duration the entire strip remains lit after the last
//     trigger.
//
// 2. Animation Cycle (State Machine):
// The core logic is implemented as a state machine orchestrated by the runner
// function. It cycles through three distinct states, each managed by a
// dedicated helper function:
//   - State 1: Run-Up (runUpPhase)
//     The animation starts at ledIndex and turns on LEDs symmetrically outwards
//     from the center (left--, right++) until the light covers both ends of
//     the LED strip.
//   - State 2: Hold (holdPhase)
//     Once fully lit, the strip enters a "hold" state for the configured
//     holdT duration. If a new trigger arrives, the hold timer is reset,
//     extending the animation and making the system feel responsive.
//   - State 3: Run-Down (runDownPhase)
//     After the hold time expires, LEDs turn off from the outer edges inwards
//     (left++, right--). If a new trigger arrives during this phase, the state
//     machine transitions back to the "Run-Up" state from the current light
//     position, ensuring the strip relights quickly.
//
// 3. Termination:
// The runner goroutine, and thus the animation, terminates only when the
// "Run-Down" phase completes fully without being interrupted by a new trigger.
// Once it ends, the producer becomes idle, ready to be started again by a
// future sensor event.

package producer

import (
	t "time"

	u "lautenbacher.net/goleds/util"
)

type SensorLedProducer struct {
	*AbstractProducer
	ledIndex int
	holdT    t.Duration
	runUpT   t.Duration
	runDownT t.Duration
	ledOn    Led
}

func NewSensorLedProducer(uid string, index int, ledsChanged *u.AtomicEvent[LedProducer], ledsTotal int, holdT t.Duration, runUpT t.Duration, runDownT t.Duration, ledRGB []float64) *SensorLedProducer {
	inst := &SensorLedProducer{
		ledIndex: index,
		holdT:    holdT,
		runUpT:   runUpT,
		runDownT: runDownT,
		ledOn: Led{
			Red:   ledRGB[0],
			Green: ledRGB[1],
			Blue:  ledRGB[2],
		},
	}
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	return inst
}

// runUpPhase handles the "run-up" part of the animation, where LEDs
// are turned on from the center outwards.
func (s *SensorLedProducer) runUpPhase(left, right int) (nleft, nright int, stopped bool) {
	ticker := t.NewTicker(s.runUpT)
	defer ticker.Stop()

	for {
		if left >= 0 {
			s.setLed(left, s.ledOn)
		}
		if right < len(s.leds) {
			s.setLed(right, s.ledOn)
		}
		s.ledsChanged.Send(s)

		if left <= 0 && right >= len(s.leds)-1 {
			// run-up is complete
			return left, right, false
		}

		left--
		right++

		select {
		case <-ticker.C:
		case <-s.stop:
			return left, right, true
		}
	}
}

// holdPhase handles the "hold" part, keeping LEDs on. The hold time
// is extended if new triggers arrive.
func (s *SensorLedProducer) holdPhase() (lastStartSeen t.Time, stopped bool) {
	for {
		lastStart := s.getLastTrigger().Timestamp
		holdUntil := lastStart.Add(s.holdT)

		if t.Now().After(holdUntil) {
			// Hold time expired
			return lastStart, false
		}

		select {
		case <-t.After(t.Until(holdUntil)):
			// Time expired, loop again to re-check for new triggers
		case <-s.stop:
			return t.Time{}, true
		}
	}
}

// runDownPhase handles the "run-down" part, turning LEDs off from the
// edges inwards. It can be interrupted by a new trigger, which
// signals that the animation should restart.
func (s *SensorLedProducer) runDownPhase(left, right int, lastStartSeen t.Time) (nleft, nright int, shouldRestart, stopped bool) {
	ticker := t.NewTicker(s.runDownT)
	defer ticker.Stop()

	for {
		if s.getLastTrigger().Timestamp.After(lastStartSeen) {
			// New trigger arrived, restart animation cycle
			return left, right, true, false
		}

		if left <= s.ledIndex && left >= 0 {
			s.setLed(left, Led{})
		}
		if right >= s.ledIndex && right < len(s.leds) {
			s.setLed(right, Led{})
		}
		s.ledsChanged.Send(s)

		if left == s.ledIndex && right == s.ledIndex {
			// Run-down complete, final check for new trigger
			if s.getLastTrigger().Timestamp.After(lastStartSeen) {
				return left, right, true, false // restart
			}
			return left, right, false, false // normal exit
		}

		left++
		right--

		select {
		case <-ticker.C:
		case <-s.stop:
			return left, right, false, true
		}
	}
}

// The main worker, doing a run-up, hold, and run-down cycle (if
// undisturbed by intermediate Start() events). It checks for these
// intermediate Start() events during hold time (to prolong the hold
// time accordingly) and during run-down to switch back into the
// run-up part if needed. At the end it checks one last time for an
// intermediate Start() before finally setting s.isRunning to false and
// ending the go routine. All this is either guarded directly or
// indirectly (by calls to s.getLastStart()) by s.updateMutex.
func (s *SensorLedProducer) runner(trigger *u.Trigger) {
	defer s.setIsRunning(false)

	left, right := s.ledIndex, s.ledIndex

	for {
		var stopped, shouldRestart bool
		var lastStartSeen t.Time

		left, right, stopped = s.runUpPhase(left, right)
		if stopped {
			return
		}

		lastStartSeen, stopped = s.holdPhase()
		if stopped {
			return
		}

		left, right, shouldRestart, stopped = s.runDownPhase(left, right, lastStartSeen)
		if stopped {
			return
		}

		if !shouldRestart {
			// Animation finished normally
			return
		}
		// A new trigger arrived during run-down, so restart the cycle.
	}
}
