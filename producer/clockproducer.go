package producer

import (
	"log/slog"
	"math"
	"time"

	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

type ClockProducer struct {
	*AbstractProducer
	hour         Led
	minute       Led
	hour_dist    float64
	minute_dist  float64
	hour_start   int
	minute_start int
}

func NewClockProducer(uid string, ledsChanged *u.AtomicMapEvent[LedProducer], ledsTotal int, cfg c.ClockLEDConfig) *ClockProducer {
	hour_start := cfg.StartLedHour
	hour_end := cfg.EndLedHour
	hour_length := hour_end - hour_start

	minute_start := cfg.StartLedMinute
	minute_end := cfg.EndLedMinute
	minute_length := minute_end - minute_start

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
		hour_dist:    float64(hour_length) / (12*60.0 - 1),
		minute_dist:  float64(minute_length) / (60.0 - 1),
		hour_start:   hour_start,
		minute_start: minute_start,
	}
	slog.Debug("Clock distances", "hour_dist", inst.hour_dist, "minute_dist", inst.minute_dist)
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	return inst
}

func (s *ClockProducer) setTime() {
	s.ledsMutex.Lock()
	clear(s.leds)
	now := time.Now()
	hour := now.Hour() % 12
	minute := now.Minute()
	hIdx := s.hour_start + int(math.Round(float64(hour*60+minute)*s.hour_dist))
	mIdx := s.minute_start + int(math.Round(float64(minute)*s.minute_dist))
	if hIdx >= 0 && hIdx < len(s.leds) {
		s.leds[hIdx] = s.hour
	}
	if mIdx >= 0 && mIdx < len(s.leds) {
		s.leds[mIdx] = s.minute
	}
	s.ledsMutex.Unlock()
	s.ledsChanged.Send(s.GetUID(), s)
}

func (s *ClockProducer) runner() {
	defer s.ClearLeds()

	s.setTime()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.setTime()
		case <-s.stopchan:
			return
		}
	}
}
