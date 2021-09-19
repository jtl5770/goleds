package ledcontroller

import (
	"math"
	"sync"
	t "time"
)

var NULL_LED = Led{0, 0, 0}

type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Fire()
}

type Led struct {
	Red   byte
	Green byte
	Blue  byte
}

func (s Led) IsEmpty() bool {
	return s.Red == 0 && s.Green == 0 && s.Blue == 0
}

func (s Led) Intensity() byte {
	return byte(math.Round(float64(s.Red+s.Green+s.Blue) / 3.0))
}

func (s Led) Max(in Led) Led {
	if s.Red > in.Red {
		in.Red = s.Red
	}
	if s.Green > in.Green {
		in.Green = s.Green
	}
	if s.Blue > in.Blue {
		in.Blue = s.Blue
	}
	return in
}

type AbstractProducer struct {
	uid       string
	leds      []Led
	isRunning bool
	lastFire  t.Time
	// Guards getting and setting LED values
	ledsMutex sync.Mutex
	// Guards changes to lastFire & isRunning
	updateMutex sync.Mutex
	ledsChanged chan (LedProducer)
}

// Returns a slice with the current values of all the LEDs.
// Guarded by s.ledsMutex
func (s *AbstractProducer) GetLeds() []Led {
	s.ledsMutex.Lock()
	defer s.ledsMutex.Unlock()
	ret := make([]Led, len(s.leds))
	copy(ret, s.leds)
	return ret
}

func (s *AbstractProducer) GetUID() string {
	return s.uid
}

// Return the s.lastFire value, guarded by s.updateMutex
func (s *SensorLedProducer) getLastFire() t.Time {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	return s.lastFire
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
