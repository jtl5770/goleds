// this producer lights the whole strip with another (maybe brighter)
// color, when ever a sensor is triggered with a configurable (usually
// high) value for a configurable time.  It will hold this color for a
// configurable time and then switch off again. Triggering the sensor
// again for the configured duration while it is running will stop the
// producer.

package producer

import (
	"time"
	t "time"

	c "lautenbacher.net/goleds/config"
)

type HoldProducer struct {
	*AbstractProducer
	ledOnHold Led
	holdT     time.Duration
}

func NewHoldProducer(uid string, ledsChanged chan LedProducer) *HoldProducer {
	inst := HoldProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
		ledOnHold:        Led{Red: c.CONFIG.HoldLED.LedRGB[0], Green: c.CONFIG.HoldLED.LedRGB[1], Blue: c.CONFIG.HoldLED.LedRGB[2]},
		holdT:            c.CONFIG.HoldLED.HoldTime,
	}
	inst.runfunc = inst.runner
	return &inst
}

func (s *HoldProducer) runner(startime t.Time) {
	defer func() {
		for idx := range s.leds {
			s.setLed(idx, Led{})
		}
		s.ledsChanged <- s
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
	}()

	for idx := range s.leds {
		s.setLed(idx, s.ledOnHold)
	}
	s.ledsChanged <- s

	for {
		select {
		case <-s.stop:
			return
		case <-time.After(s.holdT):
			return
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
