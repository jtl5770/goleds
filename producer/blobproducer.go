package producer

import (
	"log"
	"math"
	"time"

	c "lautenbacher.net/goleds/config"
)

type BlobProducer struct {
	*AbstractProducer
	last_x float64
	x      float64
	width  float64
	led    Led
	delta  float64
	dir    float64
}

func NewBlobProducer(uid string, ledsChanged chan LedProducer) *BlobProducer {
	inst := BlobProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
		led: Led{
			Red:   c.CONFIG.BlobLED.BlobCfg[uid].LedRGB[0],
			Green: c.CONFIG.BlobLED.BlobCfg[uid].LedRGB[1],
			Blue:  c.CONFIG.BlobLED.BlobCfg[uid].LedRGB[2],
		},
		last_x: c.CONFIG.BlobLED.BlobCfg[uid].X,
		x:      c.CONFIG.BlobLED.BlobCfg[uid].X,
		width:  c.CONFIG.BlobLED.BlobCfg[uid].Width,
	}
	inst.runfunc = inst.runner
	inst.delta = c.CONFIG.BlobLED.BlobCfg[uid].DeltaX
	if inst.delta < 0 {
		inst.dir = -1
	} else {
		inst.dir = 1
	}
	inst.delta = math.Abs(inst.delta)
	return &inst
}

func (s *BlobProducer) getMovement() (float64, float64) {
	old := s.last_x
	cur := s.x
	s.last_x = cur
	return old, cur
}

func (s *BlobProducer) toggleDir() {
	s.dir = s.dir * -1
}

func (s *BlobProducer) runner() {
	defer func() {
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
	}()

	tickX := time.NewTicker(c.CONFIG.BlobLED.BlobCfg[s.uid].Delay)
	for {
		for i := 0; i < c.CONFIG.Hardware.Display.LedsTotal; i++ {
			y := math.Exp(-1 * (math.Pow(float64(i)-s.x, 2) / s.width))
			s.setLed(i, Led{byte(math.Round(float64(s.led.Red) * y)), byte(math.Round(float64(s.led.Green) * y)), byte(math.Round(float64(s.led.Blue) * y))})
		}
		s.ledsChanged <- s

		select {
		case <-s.stop:
			log.Println("Stopped BlobProducer...")
			tickX.Stop()
			return
		case <-tickX.C:
			s.x = s.x + (s.delta * s.dir)
		}
	}
}

func DetectCollisions(prods [](*BlobProducer), sig chan bool) {
	max := float64(c.CONFIG.Hardware.Display.LedsTotal)
	tick := time.NewTicker(50 * time.Millisecond)

	for {
		select {
		case <-tick.C:
			var inter [](*BlobProducer)
			// detect reaching beginning or end of stripe
			for _, prod := range prods {
				if ((prod.x > max) && (prod.dir > 0)) ||
					((prod.x < 0) && (prod.dir < 0)) {
					prod.toggleDir()
				} else {
					inter = append(inter, prod)
				}
			}
			size := len(inter)
			if size >= 2 {
				for i := 0; i < size; i++ {
					prod_a := inter[i]
					for j := i + 1; j < size; j++ {
						prod_b := inter[j]
						detectIntra(prod_a, prod_b)
					}
				}
			}
		case <-sig:
			log.Println("Ending detectCollisions go-routine")
			tick.Stop()
			return
		}
	}
}

func detectIntra(prod_a *BlobProducer, prod_b *BlobProducer) bool {
	a1, a2 := prod_a.getMovement()
	a_start := math.Min(a1, a2)
	a_end := math.Max(a1, a2)
	b1, b2 := prod_b.getMovement()
	b_start := math.Min(b1, b2)
	b_end := math.Max(b1, b2)
	return (a_start <= b_end) && (b_start <= a_end)
}
