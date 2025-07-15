package platform

import (
	"sync"
	"time"

	p "lautenbacher.net/goleds/producer"
)

// Platform defines the interface for abstracting away the real
// hardware or the TUI simulation. The rest of the program should only
// see this interface.
type Platform interface {
	// Start initializes the platform (e.g., opens GPIO/SPI, or starts the TUI).
	Start() error

	// Stop cleans up all platform resources.
	Stop()

	// DisplayLeds sends the complete state of all LEDs to the output device.
	// This will be either the LED stripes or the TUI simulation
	DisplayLeds(leds []p.Led)

	// GetSensorEvents returns a channel that the application can read from
	// to receive sensor trigger events.
	GetSensorEvents() <-chan *Trigger

	// GetSensorLedIndices returns a map of sensor UIDs to their LED indices.
	GetSensorLedIndices() map[string]int

	// GetLedsTotal returns the total number of configured LEDs.
	GetLedsTotal() int

	// ForceUpdateDelay returns the configured delay for forcing a display update.
	GetForceUpdateDelay() time.Duration

	// DisplayDriver runs the display update loop for the platform.
	DisplayDriver(display chan []p.Led, stopSignal chan bool, wg *sync.WaitGroup)

	// SensorDriver runs the sensor reading loop for the platform.
	SensorDriver(stopSignal chan bool, wg *sync.WaitGroup)
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
