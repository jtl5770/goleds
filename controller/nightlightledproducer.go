package ledcontroller

import (
	t "time"

	"github.com/nathan-osman/go-sunrise"
)

type NightlightLedProducter struct {
	AbstractProducer
	latitude  float64
	longitude float64
	ledNight  Led
}

func NewNightlightLedProducter(uid string, size int, ledsChanged chan (LedProducer),
	ledNight Led, latitude float64, longitude float64) *NightlightLedProducter {
	leds := make([]Led, size)
	inst := &NightlightLedProducter{
		AbstractProducer: AbstractProducer{
			leds:        leds,
			uid:         uid,
			isRunning:   false,
			ledsChanged: ledsChanged},
		latitude:  latitude,
		longitude: longitude,
		ledNight:  ledNight}
	inst.runfunc = inst.runner
	return inst
}

func (s *NightlightLedProducter) setLed(on bool) {
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

func (s *NightlightLedProducter) runner() {
	for {
		now := t.Now()
		next := now.Add(24 * t.Hour) // tomorrow
		rise, set := sunrise.SunriseSunset(s.latitude, s.longitude, now.Year(), now.Month(), now.Day())
		rise_next, _ := sunrise.SunriseSunset(s.latitude, s.longitude, next.Year(), next.Month(), next.Day())
		if now.After(rise) && now.Before(set) {
			// During the day - between sunrise and sunset
			s.setLed(false)
			s.ledsChanged <- s
			t.Sleep(set.Sub(now))
		} else if now.Before(rise) {
			// in the night after midnight but before sunrise
			s.setLed(true)
			s.ledsChanged <- s
			t.Sleep(rise.Sub(now))
		} else if now.Before(rise_next) {
			// in the night before midnight
			s.setLed(true)
			s.ledsChanged <- s
			t.Sleep(rise_next.Sub(now))
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
