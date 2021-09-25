package producer

import (
	"log"
	"time"

	c "lautenbacher.net/goleds/config"

	"github.com/nathan-osman/go-sunrise"
)

type NightlightProducer struct {
	*AbstractProducer
	latitude  float64
	longitude float64
	ledNight  Led
}

func NewNightlightProducer(uid string, ledsChanged chan (LedProducer)) *NightlightProducer {
	inst := NightlightProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
		latitude:         c.CONFIG.NightLED.Latitude,
		longitude:        c.CONFIG.NightLED.Longitude,
		ledNight:         Led{Red: c.CONFIG.NightLED.LedRed, Green: c.CONFIG.NightLED.LedGreen, Blue: c.CONFIG.NightLED.LedBlue}}
	inst.runfunc = inst.runner
	return &inst
}

func (s *NightlightProducer) setLed(on bool) {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	if on {
		for i := range s.leds {
			s.leds[i] = s.ledNight
		}
	} else {
		for i := range s.leds {
			s.leds[i] = Led{}
		}
	}
}

func (s *NightlightProducer) runner() {
	defer func() {
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
	}()

	for {
		now := time.Now()
		next := now.Add(24 * time.Hour) // tomorrow
		rise, set := sunrise.SunriseSunset(s.latitude, s.longitude, now.Year(), now.Month(), now.Day())
		rise_next, _ := sunrise.SunriseSunset(s.latitude, s.longitude, next.Year(), next.Month(), next.Day())
		var wakeupAfter time.Duration
		if now.After(rise) && now.Before(set) {
			// During the day - between sunrise and sunset
			s.setLed(false)
			s.ledsChanged <- s
			wakeupAfter = set.Sub(now)
		} else if now.Before(rise) {
			// in the night after midnight but before sunrise
			s.setLed(true)
			s.ledsChanged <- s
			wakeupAfter = rise.Sub(now)
		} else {
			// in the night before midnight - need to sleep unit rise_next
			s.setLed(true)
			s.ledsChanged <- s
			wakeupAfter = rise_next.Sub(now)
		}
		select {
		case <-time.After(wakeupAfter):
			// nothing, just continue
		case <-s.stop:
			log.Println("Stopped NightlightProducer...")
			return
		}
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
