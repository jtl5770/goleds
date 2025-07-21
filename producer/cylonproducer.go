package producer

import (
	"math"
	"time"

	u "lautenbacher.net/goleds/util"
)

type CylonProducer struct {
	*AbstractProducer
	x         float64
	step      float64
	radius    int
	direction int
	color     Led
	duration  time.Duration
	delay     time.Duration
}

func NewCylonProducer(uid string, ledsChanged *u.AtomicEvent[LedProducer], ledsTotal int, duration time.Duration, delay time.Duration, step float64, width int, ledRGB []float64) *CylonProducer {
	inst := &CylonProducer{
		color: Led{
			Red:   ledRGB[0],
			Green: ledRGB[1],
			Blue:  ledRGB[2],
		},
		step:      step,
		x:         0,
		direction: 1,
		duration:  duration,
		delay:     delay,
	}
	inst.radius = width / 2
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)

	return inst
}

func (s *CylonProducer) runner(trigger *u.Trigger) {
	triggerduration := time.NewTicker(s.duration)
	tick := time.NewTicker(s.delay)
	defer func() {
		for i := range s.leds {
			s.setLed(i, Led{})
		}
		s.ledsChanged.Send(s)
		tick.Stop()
		triggerduration.Stop()
		s.setIsRunning(false)
	}()

	for {
		select {
		case <-triggerduration.C:
			return
		case <-s.stop:
			return
		case <-tick.C:
			if s.x < 0 || s.x > float64(len(s.leds)-1) {
				s.direction = -s.direction
			}
			s.x += float64(s.direction) * s.step
			left := s.x - float64(s.radius)
			right := s.x + float64(s.radius)
			// log.Printf("x: %f, left: %f, right: %f\n", s.x, left, right)
			for i := range s.leds {
				if i < int(left) || i > int(right+1) {
					s.setLed(i, Led{})
				} else {
					if i == int(math.Floor(left)) {
						f := 1 - (left - float64(i))
						s.setLed(i, Led{s.color.Red * f, s.color.Green * f, s.color.Blue * f})
					} else if i == int(math.Floor(right+1)) {
						f := 1 - (float64(i) - right)
						s.setLed(i, Led{s.color.Red * f, s.color.Green * f, s.color.Blue * f})
					} else {
						s.setLed(i, s.color)
					}
				}
			}
			s.ledsChanged.Send(s)
		}
	}
}
