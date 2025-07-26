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
	"log"
	"sync"
	t "time"

	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

type SensorLedProducer struct {
	*AbstractProducer
	ledIndex          int
	holdT             t.Duration
	runUpT            t.Duration
	runDownT          t.Duration
	ledOn             Led
	latchEnabled      bool
	latchTriggerValue int
	latchTriggerDelay t.Duration
	latchTime         t.Duration
	latchLed          Led
}

func NewSensorLedProducer(uid string, index int, ledsChanged *u.AtomicEvent[LedProducer], ledsTotal int, cfg c.SensorLEDConfig, endwg *sync.WaitGroup) *SensorLedProducer {
	inst := &SensorLedProducer{
		ledIndex:          index,
		holdT:             cfg.HoldTime,
		runUpT:            cfg.RunUpDelay,
		runDownT:          cfg.RunDownDelay,
		latchEnabled:      cfg.LatchEnabled,
		latchTriggerValue: cfg.LatchTriggerValue,
		latchTriggerDelay: cfg.LatchTriggerDelay,
		latchTime:         cfg.LatchTime,
		ledOn: Led{
			Red:   cfg.LedRGB[0],
			Green: cfg.LedRGB[1],
			Blue:  cfg.LedRGB[2],
		},
		latchLed: Led{
			Red:   cfg.LatchLedRGB[0],
			Green: cfg.LatchLedRGB[1],
			Blue:  cfg.LatchLedRGB[2],
		},
	}
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	if endwg != nil {
		inst.AbstractProducer.endWg = endwg
	}
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
		case <-s.stopchan:
			return left, right, true
		}
	}
}

// is extended if new triggers arrive. It also checks for the "latch"
// trigger pattern.
func (s *SensorLedProducer) holdPhase() (stopped bool) {
	var latchStart t.Time
	inLatchZone := false

	for {
		holdTimer := t.NewTimer(s.holdT)

		select {
		case <-s.stopchan:
			holdTimer.Stop()
			return true // Stop requested
		case <-holdTimer.C:
			return false // Hold time expired
		case <-s.triggerEvent.Channel():
			holdTimer.Stop() // Reset hold timer on any trigger
			trigger := s.triggerEvent.Value()

			if s.latchEnabled && trigger.Value >= s.latchTriggerValue {
				if !inLatchZone {
					// Start of a potential latch-on sequence
					inLatchZone = true
					latchStart = trigger.Timestamp
				} else {
					// Check if the latch-on delay has been met
					if t.Since(latchStart) >= s.latchTriggerDelay {
						if s.runLatchMode() {
							return true // Latch mode was stopped via stopchan
						}
						// Latch mode finished, reset and continue normal hold.
						inLatchZone = false
					}
				}
			} else {
				// Trigger was not a latch trigger, reset the sequence.
				inLatchZone = false
			}
		}
	}
}

// runLatchMode activates the high-intensity "latch" mode. It remains
// active for latchTime unless another latch trigger toggles it off early.
func (s *SensorLedProducer) runLatchMode() (stopped bool) {
	log.Printf("   ===> Latch Mode Activated for %s", s.GetUID())
	// Set all LEDs to the bright latch color
	for i := range s.leds {
		s.setLed(i, s.latchLed)
	}
	s.ledsChanged.Send(s)

	// Defer reverting the LEDs to the normal color to simplify exit paths.
	defer func() {
		for i := range s.leds {
			s.setLed(i, s.ledOn)
		}
		s.ledsChanged.Send(s)
	}()

	latchTimer := t.NewTimer(s.latchTime)
	defer latchTimer.Stop()

	var latchOffStart t.Time
	inLatchOffZone := false

	for {
		select {
		case <-s.stopchan:
			return true // Stop requested by system
		case <-latchTimer.C:
			// Main latch time expired
			log.Printf("   <=== Latch Mode Timed Out for %s", s.GetUID())
			return false
		case <-s.triggerEvent.Channel():
			trigger := s.triggerEvent.Value()
			if s.latchEnabled && trigger.Value >= s.latchTriggerValue {
				if !inLatchOffZone {
					// Start of a potential latch-off sequence
					inLatchOffZone = true
					latchOffStart = trigger.Timestamp
				} else {
					// Check if the latch-off delay has been met
					if t.Since(latchOffStart) >= s.latchTriggerDelay {
						log.Printf("   <=== Latch Mode Deactivated by toggle for %s", s.GetUID())
						return false
					}
				}
			} else {
				// Not a latch trigger, reset the toggle-off sequence.
				inLatchOffZone = false
			}
		}
	}
}

// runDownPhase handles the "run-down" part, turning LEDs off from the
// edges inwards. It can be interrupted by a new trigger, which
// signals that the animation should restart.
func (s *SensorLedProducer) runDownPhase(left, right int) (nleft, nright int, shouldRestart, stopped bool) {
	ticker := t.NewTicker(s.runDownT)
	defer ticker.Stop()
	for {
		if left <= s.ledIndex && left >= 0 {
			s.setLed(left, Led{})
		}
		if right >= s.ledIndex && right < len(s.leds) {
			s.setLed(right, Led{})
		}
		s.ledsChanged.Send(s)
		if left == s.ledIndex && right == s.ledIndex {
			return left, right, false, false // normal exit
		}
		if left < s.ledIndex {
			left++
		}
		if right > s.ledIndex {
			right--
		}

		select {
		case <-s.triggerEvent.Channel():
			// New trigger arrived, restart animation cycle
			return left, right, true, false
		case <-s.stopchan:
			return left, right, false, true // Stop requested
		case <-ticker.C:
			// Continue run-down phase
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
func (s *SensorLedProducer) runner() {
	defer log.Printf("   <=== Stopping SensorLedProducer %s", s.GetUID())

	select {
	case <-s.triggerEvent.Channel():
		left, right := s.ledIndex, s.ledIndex
		for {
			var stopped, shouldRestart bool

			left, right, stopped = s.runUpPhase(left, right)
			if stopped {
				return
			}

			stopped = s.holdPhase()
			if stopped {
				return
			}

			left, right, shouldRestart, stopped = s.runDownPhase(left, right)
			if stopped {
				return
			}

			if !shouldRestart {
				// Animation finished normally, exit the cycle
				return
			}
			// A new trigger arrived during run-down, so restart animation.
		}
	case <-s.stopchan:
		return
	}
}
