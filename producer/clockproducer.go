package producer

import (
	"log/slog"
	"math"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/util"
)

type ClockProducer struct {
	*AbstractProducer
	hour        Led
	minute      Led
	hourDist    float64
	minuteDist  float64
	hourStart   int
	minuteStart int
}

func NewClockProducer(uid string, ledsChanged *util.AtomicMapEvent[LedProducer], ledsTotal int, cfg config.ClockLEDConfig) *ClockProducer {
	hourStart := cfg.StartLedHour
	hourEnd := cfg.EndLedHour
	hourLength := hourEnd - hourStart

	minuteStart := cfg.StartLedMinute
	minuteEnd := cfg.EndLedMinute
	minuteLength := minuteEnd - minuteStart

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
		hourDist:    float64(hourLength) / (12*60.0 - 1),
		minuteDist:  float64(minuteLength) / (60.0 - 1),
		hourStart:   hourStart,
		minuteStart: minuteStart,
	}
	slog.Debug("Clock distances", "hourDist", inst.hourDist, "minuteDist", inst.minuteDist)
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	return inst
}

func (s *ClockProducer) setTime() {
	s.ledsMutex.Lock()
	clear(s.leds)
	now := time.Now()
	hour := now.Hour() % 12
	minute := now.Minute()
	hIdx := s.hourStart + int(math.Round(float64(hour*60+minute)*s.hourDist))
	mIdx := s.minuteStart + int(math.Round(float64(minute)*s.minuteDist))
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
