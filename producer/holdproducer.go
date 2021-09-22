package producer

import "time"

type HoldProducer struct {
	AbstractProducer
	ledOnHold Led
	holdT     time.Duration
}

func NewHoldProducer(uid string, size int, ledsChanged chan (LedProducer),
	ledOnHold Led, holdT time.Duration) *HoldProducer {
	leds := make([]Led, size)
	inst := &HoldProducer{
		AbstractProducer: AbstractProducer{
			leds:        leds,
			uid:         uid,
			isRunning:   false,
			ledsChanged: ledsChanged},
		ledOnHold: ledOnHold,
		holdT:     holdT}
	inst.runfunc = inst.runner
	return inst
}

func (s *HoldProducer) runner() {
	for idx := range s.leds {
		s.setLed(idx, s.ledOnHold)
	}
	s.ledsChanged <- s
	time.Sleep(s.holdT)
	for idx := range s.leds {
		s.setLed(idx, NULL_LED)
	}
	s.ledsChanged <- s
}
