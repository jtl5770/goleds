package producer

import (
	"log"
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
	ticker := time.NewTicker(time.Second)
	defer func() {
		ticker.Stop()
		for idx := range s.leds {
			s.setLed(idx, Led{})
		}
		s.ledsChanged <- s
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
	}()

	initial := s.getLastStart()
	for idx := range s.leds {
		s.setLed(idx, s.ledOnHold)
	}
	s.ledsChanged <- s

	for {
		select {
		case <-s.stop:
			log.Println("Stopped HoldProducer...")
			return
		case <-ticker.C:
			if (time.Now().Sub(initial) >= s.holdT) || s.getLastStart().After(initial) {
				return
			}
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
