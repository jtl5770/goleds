package producer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	u "lautenbacher.net/goleds/util"
)

func TestNewHoldProducer(t *testing.T) {
	ledsChanged := u.NewAtomicEvent[LedProducer]()
	ledsTotal := 10
	holdTime := 100 * time.Millisecond
	ledRGB := []float64{1, 2, 3}

	p := NewHoldProducer("test", ledsChanged, ledsTotal, holdTime, ledRGB)

	assert.Equal(t, "test", p.GetUID())
	assert.Equal(t, holdTime, p.holdT)
	assert.Equal(t, Led{Red: 1, Green: 2, Blue: 3}, p.ledOnHold)
	assert.Len(t, p.leds, ledsTotal)
	assert.False(t, p.GetIsRunning())
}

func TestHoldProducer_Runner(t *testing.T) {
	ledsChanged := u.NewAtomicEvent[LedProducer]()
	ledsTotal := 5
	holdTime := 50 * time.Millisecond
	ledRGB := []float64{255, 0, 0}

	p := NewHoldProducer("test", ledsChanged, ledsTotal, holdTime, ledRGB)

	// Start the producer
	p.Start(u.NewTrigger("test", 0, time.Now()))
	time.Sleep(10 * time.Millisecond) // Give runner time to start and set LEDs

	// Check that it's running and LEDs are on
	assert.True(t, p.GetIsRunning())
	leds := p.GetLeds()
	for _, led := range leds {
		assert.Equal(t, Led{Red: 255, Green: 0, Blue: 0}, led)
	}

	// Wait for hold time to expire
	time.Sleep(holdTime)

	// Check that it's stopped and LEDs are off
	assert.False(t, p.GetIsRunning())
	leds = p.GetLeds()
	for _, led := range leds {
		assert.True(t, led.IsEmpty())
	}
}

func TestHoldProducer_Stop(t *testing.T) {
	ledsChanged := u.NewAtomicEvent[LedProducer]()
	ledsTotal := 5
	holdTime := 200 * time.Millisecond
	ledRGB := []float64{255, 0, 0}

	p := NewHoldProducer("test", ledsChanged, ledsTotal, holdTime, ledRGB)

	p.Start(u.NewTrigger("test", 0, time.Now()))
	time.Sleep(10 * time.Millisecond)
	assert.True(t, p.GetIsRunning())

	p.Stop()
	time.Sleep(10 * time.Millisecond) // Give runner time to process stop signal

	assert.False(t, p.GetIsRunning())
	leds := p.GetLeds()
	for _, led := range leds {
		assert.True(t, led.IsEmpty())
	}
}
