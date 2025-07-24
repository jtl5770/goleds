// This producer displays a constant color on the stripes between
// sunset and sunrise. There can be different colors for different
// times of the night. The colors are configured in the config file.

package producer

import (
	"time"

	u "lautenbacher.net/goleds/util"

	"github.com/nathan-osman/go-sunrise"
)

type NightlightProducer struct {
	*AbstractProducer
	latitude  float64
	longitude float64
	ledNight  []Led
}

func NewNightlightProducer(uid string, ledsChanged *u.AtomicEvent[LedProducer], ledsTotal int, latitude float64, longitude float64, ledRGB [][]float64) *NightlightProducer {
	inst := &NightlightProducer{
		latitude:  latitude,
		longitude: longitude,
		ledNight:  make([]Led, len(ledRGB)),
	}
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	for index, led := range ledRGB {
		inst.ledNight[index] = Led{led[0], led[1], led[2]}
	}
	return inst
}

func (s *NightlightProducer) setNightLed(index int) {
	for i := range s.leds {
		s.setLed(i, s.ledNight[index])
	}
}

func (s *NightlightProducer) runner() {
	defer func() {
		s.leds = make([]Led, len(s.leds)) // Reset LEDs
		s.ledsChanged.Send(s)
	}()

	for {
		now := time.Now()
		next := now.Add(24 * time.Hour)  // tomorrow
		prev := now.Add(-24 * time.Hour) // yesterday
		rise, set := sunrise.SunriseSunset(s.latitude, s.longitude, now.Year(), now.Month(), now.Day())
		rise_next_day, _ := sunrise.SunriseSunset(s.latitude, s.longitude, next.Year(), next.Month(), next.Day())
		_, set_prev_day := sunrise.SunriseSunset(s.latitude, s.longitude, prev.Year(), prev.Month(), prev.Day())
		var wakeupAfter time.Duration
		if now.After(rise) && now.Before(set) {
			// During the day - between sunrise and sunset
			s.leds = make([]Led, len(s.leds)) // Reset LEDs
			s.ledsChanged.Send(s)
			wakeupAfter = set.Sub(now)
		} else {
			var waitIntervalDuration time.Duration
			var tillNextInterval time.Duration
			var currInterval int
			if now.Before(rise) {
				// in the night after midnight but before sunrise.
				// The "total" night duration is this days sunrise -
				// previous days sunset The lenght that each
				// configured LED value should be used is computed by
				// dividing the night duration by the number of
				// configured night LED Konfigurations
				waitIntervalDuration = time.Duration(rise.Sub(set_prev_day).Nanoseconds() / int64(len(s.ledNight)))
				currInterval = int(now.Sub(set_prev_day) / waitIntervalDuration)
				tillNextInterval = set_prev_day.Add(time.Duration((currInterval + 1)) * waitIntervalDuration).Sub(now)
			} else {
				// in the night before midnight - similar as above but
				// using current days sunset and next days sunrise
				waitIntervalDuration = time.Duration(rise_next_day.Sub(set).Nanoseconds() / int64(len(s.ledNight)))
				currInterval = int(now.Sub(set) / waitIntervalDuration)
				tillNextInterval = set.Add(time.Duration((currInterval + 1)) * waitIntervalDuration).Sub(now)
			}
			// log.Printf("Current NightLED index %d : waitInterval %d : tillNextInterval %d", currInterval, waitIntervalDuration, tillNextInterval)
			s.setNightLed(currInterval)
			s.ledsChanged.Send(s)
			// + 1s maybe not needed, but so we are sure to really be
			// in the next interval
			wakeupAfter = tillNextInterval + time.Second
		}
		select {
		case <-time.After(wakeupAfter):
			// nothing, just continue
		case <-s.stopchan:
			// log.Println("Stopped NightlightProducer...")
			return
		}
	}
}
