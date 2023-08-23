package producer

import (
	"time"
	t "time"

	c "lautenbacher.net/goleds/config"
)

type CylonProducer struct {
	*AbstractProducer
	x         float64
	step      float64
	width     int
	direction int
}

func NewCylonProducer(uid string, ledsChanged chan LedProducer) *CylonProducer {
	inst := CylonProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
	}
	inst.x = 0
	inst.direction = 1
	inst.step = c.CONFIG.CylonLED.Step
	inst.width = c.CONFIG.CylonLED.Width
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
			s.ledsChanged <- s
		}
	}
}
