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

func NewHoldProducer(uid string, size int, ledsChanged chan (LedProducer)) *HoldProducer {
	leds := make([]Led, size)
	inst := &HoldProducer{
		AbstractProducer: AbstractProducer{
			leds:        leds,
			uid:         uid,
			isRunning:   false,
			ledsChanged: ledsChanged},
		ledOnHold: Led{Red: CONFIG.HoldLED.LedRed, Green: CONFIG.HoldLED.LedGreen, Blue: CONFIG.HoldLED.LedBlue},
		holdT:     CONFIG.HoldLED.HoldMinutes * time.Second}
	inst.runfunc = inst.runner
	return inst
}

func (s *HoldProducer) Fire() {
	log.Print(".... in Fire ....")
	s.AbstractProducer.Fire()
}

func (s *HoldProducer) runner() {
	initial := s.getLastFire()
	for idx := range s.leds {
		s.setLed(idx, s.ledOnHold)
	}
	s.ledsChanged <- s
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
		if (time.Now().Sub(initial) >= s.holdT) || s.getLastFire().After(initial) {
			ticker.Stop()
			break
		}
	}
	for idx := range s.leds {
		s.setLed(idx, NULL_LED)
	}
	s.ledsChanged <- s
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()
	s.isRunning = false
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
