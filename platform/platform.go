package platform

import (
	"time"

	p "lautenbacher.net/goleds/producer"
)

// Platform defines the interface for abstracting away the real hardware
// from the TUI simulation.
type Platform interface {
	// Start initializes the platform (e.g., opens GPIO/SPI, or starts the TUI).
	Start() error

	// Stop cleans up all platform resources.
	Stop()

	// DisplayLeds sends the complete state of all LEDs to the output device.
	DisplayLeds(leds []p.Led)

	// GetSensorEvents returns a channel that the application can read from
	// to receive sensor trigger events.
	GetSensorEvents() <-chan *Trigger
}

// Trigger represents a sensor event.
type Trigger struct {
	ID        string
	Value     int
	Timestamp time.Time
}

// NewTrigger creates a new Trigger instance.
func NewTrigger(id string, value int, time time.Time) *Trigger {
	inst := Trigger{
		ID:        id,
		Value:     value,
		Timestamp: time,
	}
	return &inst
}
