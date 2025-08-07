package producer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	u "lautenbacher.net/goleds/util"
)

func TestNewCylonProducer(t *testing.T) {
	ledsChanged := u.NewAtomicMapEvent[LedProducer]()
	p := NewCylonProducer("test", ledsChanged, 10, 1*time.Second, 10*time.Millisecond, 0.5, 4, []float64{1, 2, 3}, nil)

	assert.Equal(t, "test", p.GetUID())
	assert.Len(t, p.leds, 10)
	assert.Equal(t, 1*time.Second, p.duration)
	assert.Equal(t, 10*time.Millisecond, p.delay)
	assert.Equal(t, 0.5, p.step)
	assert.Equal(t, 2, p.radius) // width / 2
	assert.Equal(t, Led{Red: 1, Green: 2, Blue: 3}, p.color)

	// Initially, all LEDs should be off.
	leds := make([]Led, 10)
	p.GetLeds(leds)
	for _, led := range leds {
		assert.True(t, led.IsEmpty())
	}
}

func TestCylonProducer_Runner(t *testing.T) {
	ledsChanged := u.NewAtomicMapEvent[LedProducer]()
	p := NewCylonProducer("test", ledsChanged, 20, 100*time.Millisecond, 10*time.Millisecond, 1, 4, []float64{255, 0, 0}, nil)

	p.Start()
	time.Sleep(15 * time.Millisecond) // Allow one step to run

	leds := make([]Led, 20)
	p.GetLeds(leds)

	// After one step (x=1), the blob should be centered around index 1.
	// Radius is 2, so it affects indices from -1 to 3.
	// We expect LEDs at index 0, 1, 2, 3 to have some color.
	assert.False(t, leds[0].IsEmpty(), "leds[0] should not be empty")
	assert.False(t, leds[1].IsEmpty(), "leds[1] should not be empty")
	assert.False(t, leds[2].IsEmpty(), "leds[2] should not be empty")
	assert.True(t, leds[4].IsEmpty(), "leds[4] should be empty")

	time.Sleep(100 * time.Millisecond) // Wait for duration to expire

	p.GetLeds(leds)
	for _, led := range leds {
		assert.True(t, led.IsEmpty())
	}
}

func TestCylonProducer_Stop(t *testing.T) {
	ledsChanged := u.NewAtomicMapEvent[LedProducer]()
	p := NewCylonProducer("test", ledsChanged, 20, 500*time.Millisecond, 10*time.Millisecond, 1, 4, []float64{255, 0, 0}, nil)

	p.Start()
	time.Sleep(15 * time.Millisecond)
	// Check that some LEDs are on
	leds := make([]Led, 20)
	p.GetLeds(leds)
	assert.False(t, leds[1].IsEmpty())

	p.TryStop()
	time.Sleep(15 * time.Millisecond) // Give time for the stop to be processed

	// All LEDs should be off now
	p.GetLeds(leds)
	for _, led := range leds {
		assert.True(t, led.IsEmpty())
	}
}
