package producer

import (
	"log"
	"time"
)

type HoldProducer struct {
	AbstractProducer
	ledOnHold Led
	holdT     time.Duration
}

func NewHoldProducer(uid string, size int, ledsChanged chan LedProducer) *HoldProducer {
	inst := HoldProducer{
		AbstractProducer: *NewAbstractProducer(uid, size, ledsChanged),
		ledOnHold:        Led{Red: CONFIG.HoldLED.LedRed, Green: CONFIG.HoldLED.LedGreen, Blue: CONFIG.HoldLED.LedBlue},
		holdT:            CONFIG.HoldLED.HoldMinutes * time.Second}
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
		s.setLed(idx, NULL_LED)
	}
	s.ledsChanged <- s
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
