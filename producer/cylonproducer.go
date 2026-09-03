package producer

import (
	"math"
	"sync"
	"time"

	"lautenbacher.net/goleds/util"
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

func NewCylonProducer(uid string, ledsChanged *util.AtomicMapEvent[LedProducer], ledsTotal int, duration time.Duration, delay time.Duration, step float64, width int, ledRGB []float64, endwg *sync.WaitGroup) *CylonProducer {
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
	if endwg != nil {
		inst.AbstractProducer.endWg = endwg
	}

	return inst
}

func (s *CylonProducer) runner() {
	triggerduration := time.NewTicker(s.duration)
	tick := time.NewTicker(s.delay)
	defer func() {
		s.ClearLeds()
		tick.Stop()
		triggerduration.Stop()
	}()

	for {
		select {
		case <-triggerduration.C:
			return
		case <-s.stopchan:
			return
		case <-tick.C:
			if s.x < 0 || s.x > float64(len(s.leds)-1) {
				s.direction = -s.direction
			}
			s.x += float64(s.direction) * s.step
			left := s.x - float64(s.radius)
			right := s.x + float64(s.radius)

			s.ledsMutex.Lock()
			clear(s.leds)
			start := max(0, int(math.Floor(left)))
			end := min(len(s.leds)-1, int(math.Ceil(right+1)))
			for i := start; i <= end; i++ {
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
			s.ledsMutex.Unlock()
			s.ledsChanged.Send(s.GetUID(), s)
		}
	}
}
