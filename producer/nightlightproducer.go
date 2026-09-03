// This producer displays a constant color on the stripes between
// sunset and sunrise. There can be different colors for different
// times of the night. The colors are configured in the config file.

package producer

import (
	"time"

	"github.com/nathan-osman/go-sunrise"
	"lautenbacher.net/goleds/util"
)

type NightlightProducer struct {
	*AbstractProducer
	latitude  float64
	longitude float64
	ledNight  []Led
}

func NewNightlightProducer(uid string, ledsChanged *util.AtomicMapEvent[LedProducer], ledsTotal int, latitude float64, longitude float64, ledRGB [][]float64) *NightlightProducer {
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
	if len(s.ledNight) == 0 {
		return
	}
	index = max(0, min(index, len(s.ledNight)-1))

	s.ledsMutex.Lock()
	for i := range s.leds {
		s.leds[i] = s.ledNight[index]
	}
	s.ledsMutex.Unlock()
	s.ledsChanged.Send(s.GetUID(), s)
}

func (s *NightlightProducer) runner() {
	defer s.ClearLeds()

	if len(s.ledNight) == 0 {
		return
	}

	for {
		now := time.Now()
		next := now.Add(24 * time.Hour)  // tomorrow
		prev := now.Add(-24 * time.Hour) // yesterday
		rise, set := sunrise.SunriseSunset(s.latitude, s.longitude, now.Year(), now.Month(), now.Day())
		riseNextDay, _ := sunrise.SunriseSunset(s.latitude, s.longitude, next.Year(), next.Month(), next.Day())
		_, setPrevDay := sunrise.SunriseSunset(s.latitude, s.longitude, prev.Year(), prev.Month(), prev.Day())
		var wakeupAfter time.Duration
		if now.After(rise) && now.Before(set) {
			// During the day - between sunrise and sunset
			s.ClearLeds()
			wakeupAfter = set.Sub(now)
		} else {
			var waitIntervalDuration time.Duration
			var tillNextInterval time.Duration
			var currInterval int
			if now.Before(rise) {
				// in the night after midnight but before sunrise.
				// The "total" night duration is this days sunrise -
				// previous days sunset The length that each
				// configured LED value should be used is computed by
				// dividing the night duration by the number of
				// configured night LED configurations
				waitIntervalDuration = time.Duration(rise.Sub(setPrevDay).Nanoseconds() / int64(len(s.ledNight)))
				if waitIntervalDuration <= 0 {
					waitIntervalDuration = time.Minute
				}
				currInterval = int(now.Sub(setPrevDay) / waitIntervalDuration)
				tillNextInterval = setPrevDay.Add(time.Duration((currInterval + 1)) * waitIntervalDuration).Sub(now)
			} else {
				// in the night before midnight - similar as above but
				// using current days sunset and next days sunrise
				waitIntervalDuration = time.Duration(riseNextDay.Sub(set).Nanoseconds() / int64(len(s.ledNight)))
				if waitIntervalDuration <= 0 {
					waitIntervalDuration = time.Minute
				}
				currInterval = int(now.Sub(set) / waitIntervalDuration)
				tillNextInterval = set.Add(time.Duration((currInterval + 1)) * waitIntervalDuration).Sub(now)
			}
			currInterval = max(0, min(currInterval, len(s.ledNight)-1))
			// log.Printf("Current NightLED index %d : waitInterval %d : tillNextInterval %d", currInterval, waitIntervalDuration, tillNextInterval)
			s.setNightLed(currInterval)
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
