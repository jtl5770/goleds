package producer

import (
	"time"

	"github.com/nathan-osman/go-sunrise"
)

type NightlightProducter struct {
	AbstractProducer
	latitude  float64
	longitude float64
	ledNight  Led
}

func NewNightlightProducter(uid string, size int, ledsChanged chan (LedProducer)) *NightlightProducter {
	leds := make([]Led, size)
	inst := &NightlightProducter{
		AbstractProducer: AbstractProducer{
			leds:        leds,
			uid:         uid,
			isRunning:   false,
			ledsChanged: ledsChanged},
		latitude:  CONFIG.NightLED.Latitude,
		longitude: CONFIG.NightLED.Longitude,
		ledNight:  Led{Red: CONFIG.NightLED.LedRed, Green: CONFIG.NightLED.LedGreen, Blue: CONFIG.NightLED.LedBlue}}
	inst.runfunc = inst.runner
	return inst
}

func (s *NightlightProducter) setLed(on bool) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	if on {
		for i := range s.leds {
			s.leds[i] = s.ledNight
		}
	} else {
		for i := range s.leds {
			s.leds[i] = NULL_LED
		}
	}
}

func (s *NightlightProducter) runner() {
	for {
		now := time.Now()
		next := now.Add(24 * time.Hour) // tomorrow
		rise, set := sunrise.SunriseSunset(s.latitude, s.longitude, now.Year(), now.Month(), now.Day())
		rise_next, _ := sunrise.SunriseSunset(s.latitude, s.longitude, next.Year(), next.Month(), next.Day())
		if now.After(rise) && now.Before(set) {
			// During the day - between sunrise and sunset
			s.setLed(false)
			s.ledsChanged <- s
			time.Sleep(set.Sub(now))
		} else if now.Before(rise) {
			// in the night after midnight but before sunrise
			s.setLed(true)
			s.ledsChanged <- s
			time.Sleep(rise.Sub(now))
		} else {
			// in the night before midnight - need to sleep unit rise_next
			s.setLed(true)
			s.ledsChanged <- s
			time.Sleep(rise_next.Sub(now))
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
