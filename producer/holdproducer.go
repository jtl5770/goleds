package producer

import (
	"log"
	"time"

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
		holdT:            c.CONFIG.HoldLED.HoldMinutes * time.Minute,
	}
	inst.runfunc = inst.runner
	return &inst
}

func (s *HoldProducer) runner() {
	defer func() {
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
	}()

	initial := s.getLastFire()
	for idx := range s.leds {
		s.setLed(idx, s.ledOnHold)
	}
	s.ledsChanged <- s
	ticker := time.NewTicker(time.Second)
LOOP:
	for {
		select {
		case <-s.stop:
			ticker.Stop()
			log.Println("Stopped HoldProducer...")
			return
		case <-ticker.C:
			if (time.Now().Sub(initial) >= s.holdT) || s.getLastFire().After(initial) {
				ticker.Stop()
				break LOOP
			}
		}
	}
	for idx := range s.leds {
		s.setLed(idx, Led{})
	}
	s.ledsChanged <- s
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
