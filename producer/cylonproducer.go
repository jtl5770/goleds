package producer

import (
	"math"
	"time"
	t "time"

	c "lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/util"
)

type CylonProducer struct {
	*AbstractProducer
	x         float64
	step      float64
	radius    int
	direction int
	color     Led
}

func NewCylonProducer(uid string, ledsChanged *util.AtomicEvent[LedProducer]) *CylonProducer {
	inst := &CylonProducer{
		color: Led{
			Red:   c.CONFIG.CylonLED.LedRGB[0],
			Green: c.CONFIG.CylonLED.LedRGB[1],
			Blue:  c.CONFIG.CylonLED.LedRGB[2],
		},
		step:      c.CONFIG.CylonLED.Step,
		x:         0,
		direction: 1,
	}
	width := c.CONFIG.CylonLED.Width
	inst.radius = width / 2
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner)

	return inst
}

func (s *CylonProducer) runner(startTime t.Time) {
	triggerduration := time.NewTicker(c.CONFIG.CylonLED.Duration)
	tick := time.NewTicker(c.CONFIG.CylonLED.Delay)
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
			if s.x < 0 || s.x > float64(c.CONFIG.Hardware.Display.LedsTotal-1) {
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
