package producer

import (
	"log"
	"math"
	"time"

	c "lautenbacher.net/goleds/config"
)

// TODO: This creates weird effects on my hardware. I assume
// this is either crosstalk or generally producing too many updates
// per time. It is DISABLED in the config for that reason.
type BlobProducer struct {
	*AbstractProducer
	x     float64
	width float64
	led   Led
}

func NewBlobProducer(uid string, ledsChanged chan LedProducer) *BlobProducer {
	inst := BlobProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
		led: Led{
			Red:   c.CONFIG.BlobLED.BlobCfg[uid].LedRGB[0],
			Green: c.CONFIG.BlobLED.BlobCfg[uid].LedRGB[1],
			Blue:  c.CONFIG.BlobLED.BlobCfg[uid].LedRGB[2],
		},
		x:     c.CONFIG.BlobLED.BlobCfg[uid].X,
		width: c.CONFIG.BlobLED.BlobCfg[uid].Width,
	}
	inst.runfunc = inst.runner
	return &inst
}

func (s *BlobProducer) runner() {
	defer func() {
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
	}()

	delta := c.CONFIG.BlobLED.BlobCfg[s.uid].DeltaX
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
			if delta >= 0 {
				if s.x+delta > float64(c.CONFIG.Hardware.Display.LedsTotal) {
					delta = -delta
				}
			} else {
				if s.x+delta < 0 {
					delta = -delta
				}
			}
			s.x += delta
		}
	}
}
