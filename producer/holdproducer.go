package producer

import (
	"time"
	t "time"

	u "lautenbacher.net/goleds/util"
)

type HoldProducer struct {
	*AbstractProducer
	ledOnHold Led
	holdT     time.Duration
}

func NewHoldProducer(uid string, ledsChanged *u.AtomicEvent[LedProducer], ledsTotal int, holdTime time.Duration, ledRGB []float64) *HoldProducer {
	inst := &HoldProducer{
		ledOnHold: Led{Red: ledRGB[0], Green: ledRGB[1], Blue: ledRGB[2]},
		holdT:     holdTime,
	}
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	return inst
}

func (s *HoldProducer) runner(startime t.Time) {
	defer func() {
		for idx := range s.leds {
			s.setLed(idx, Led{})
		}
		s.ledsChanged.Send(s)
		s.setIsRunning(false)
	}()

	for idx := range s.leds {
		s.setLed(idx, s.ledOnHold)
	}
	s.ledsChanged.Send(s)

	for {
		select {
		case <-s.stop:
			return
		case <-time.After(s.holdT):
			return
		}
	}
}
