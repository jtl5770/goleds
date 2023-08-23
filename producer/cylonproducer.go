package producer

import (
	"math"
	"time"
	t "time"

	c "lautenbacher.net/goleds/config"
)

type CylonProducer struct {
	*AbstractProducer
	x         float64
	step      float64
	sidewidth int
	direction int
	color     Led
}

func NewCylonProducer(uid string, ledsChanged chan LedProducer) *CylonProducer {
	inst := CylonProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
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
	inst.sidewidth = width / 2

	inst.runfunc = inst.runner
	return &inst
}

func (s *CylonProducer) runner(startTime t.Time) {
	triggerduration := time.NewTicker(c.CONFIG.CylonLED.Duration)
	tick := time.NewTicker(c.CONFIG.CylonLED.Delay)
	defer func() {
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
		s.ledsMutex.Lock()
		for i := range s.leds {
			s.leds[i] = Led{}
		}
		s.ledsMutex.Unlock()
		s.ledsChanged <- s
		tick.Stop()
		triggerduration.Stop()
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
			s.ledsMutex.Lock()
			left := s.x - float64(s.sidewidth)
			right := s.x + float64(s.sidewidth)
			// log.Printf("x: %f, left: %f, right: %f\n", s.x, left, right)
			for i := range s.leds {
				if i < int(left) || i > int(right+1) {
					s.leds[i] = Led{}
				} else {
					if i == int(math.Floor(left)) {
						f := 1 - (left - float64(i))
						s.leds[i] = Led{s.color.Red * f, s.color.Green * f, s.color.Blue * f}
					} else if i == int(math.Floor(right+1)) {
						f := 1 - (float64(i) - right)
						s.leds[i] = Led{s.color.Red * f, s.color.Green * f, s.color.Blue * f}
					} else {
						s.leds[i] = s.color
					}
				}
			}
			s.ledsMutex.Unlock()
			s.ledsChanged <- s
		}
	}
}
