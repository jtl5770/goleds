package producer

import (
	"log"
	"math"
	"time"

	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

type ClockProducer struct {
	*AbstractProducer
	hour        Led
	minute      Led
	hour_dist   float64
	minute_dist float64
}

func NewClockProducer(uid string, ledsChanged *u.AtomicMapEvent[LedProducer], ledsTotal int, cfg c.ClockLEDConfig) *ClockProducer {
	start := cfg.StartLed
	end := cfg.EndLed

	length := end - start

	inst := &ClockProducer{
		hour: Led{
			Red:   cfg.LedHour[0],
			Green: cfg.LedHour[1],
			Blue:  cfg.LedHour[2],
		},
		minute: Led{
			Red:   cfg.LedMinute[0],
			Green: cfg.LedMinute[1],
			Blue:  cfg.LedMinute[2],
		},
		hour_dist:   float64(length) / 11.0,
		minute_dist: float64(length) / 59.0,
	}
	log.Printf("*** Clock distances: %f / %f", inst.hour_dist, inst.minute_dist)
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	return inst
}

func (s *ClockProducer) setTime() {
	clear(s.leds)
	now := time.Now()
	hour := now.Hour() % 12
	minute := now.Minute()
	s.setLed(int(math.Round(float64(hour)*s.hour_dist)), s.hour)
	s.setLed(int(math.Round(float64(minute)*s.minute_dist)), s.minute)
}

func (s *ClockProducer) runner() {
	defer func() {
		s.leds = make([]Led, len(s.leds)) // Reset LEDs
		s.ledsChanged.Send(s.GetUID(), s)
	}()

	s.setTime()
	s.ledsChanged.Send(s.GetUID(), s)

	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			s.setTime()
			s.ledsChanged.Send(s.GetUID(), s)
		case <-s.stopchan:
			// log.Println("Stopped NightlightProducer...")
			return
		}
	}
}
