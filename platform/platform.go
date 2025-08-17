package platform

import (
	"sync"
	"time"

	p "lautenbacher.net/goleds/producer"
	u "lautenbacher.net/goleds/util"
)

// Platform defines the interface for abstracting away the real
// hardware or the TUI simulation. The rest of the program should only
// see this interface.
type Platform interface {
	// Start initializes the platform and launches its internal goroutines.
	Start(pool *sync.Pool) error

	// Stop cleans up all platform resources and gracefully stops its goroutines.
	Stop()

	// GetLedWriter returns a write-only channel for the application to send
	// LED data to the platform.
	GetLedWriter() chan<- []p.Led

	// GetSensorEvents returns a channel that the application can read from
	// to receive sensor trigger events.
	GetSensorEvents() <-chan *u.Trigger

	// GetSensorLedIndices returns a map of sensor UIDs to their LED indices.
	GetSensorLedIndices() map[string]int

	// GetLedsTotal returns the total number of configured LEDs.
	GetLedsTotal() int

	// GetForceUpdateDelay returns the configured delay for forcing a display update.
	GetForceUpdateDelay() time.Duration

	// Ready returns a channel that is closed when the platform is fully
	// initialized and ready for producers to start.
	Ready() <-chan bool
}
