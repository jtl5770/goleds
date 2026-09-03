package platform

import (
	"sync"
	"time"

	"lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/util"
)

// CalibrationCurves maps sensor UIDs to their measured calibration curves.
type CalibrationCurves map[string][]CalibPoint

// IsCalibrated returns true if any calibration curves have been recorded.
func (c CalibrationCurves) IsCalibrated() bool {
	return len(c) > 0
}

// Platform defines the interface for abstracting away the real
// hardware or the TUI simulation. The rest of the program should only
// see this interface.
type Platform interface {
	// Start initializes the platform and launches its internal goroutines.
	Start(pool *sync.Pool, calibCurves CalibrationCurves) error

	// Stop cleans up all platform resources and gracefully stops its goroutines.
	Stop()

	// SetLeds provides a non-blocking way for the application to send the
	// latest LED data to the platform.
	SetLeds(leds []producer.Led)

	// GetSensorEvents returns a channel that the application can read from
	// to receive sensor trigger events.
	GetSensorEvents() <-chan *util.Trigger

	// GetSensorLedIndices returns a map of sensor UIDs to their LED indices.
	GetSensorLedIndices() map[string]int

	// GetLedsTotal returns the total number of configured LEDs.
	GetLedsTotal() int

	// GetForceUpdateDelay returns the configured delay for forcing a display update.
	GetForceUpdateDelay() time.Duration

	// Ready returns a channel that is closed when the platform is fully
	// initialized and ready for producers to start.
	Ready() <-chan bool

	// Calibrate performs dynamic sensor calibration and stores results in calibCurves.
	Calibrate(calibCurves CalibrationCurves) error

	// IsCalibrating reports whether a calibration routine is currently running.
	IsCalibrating() bool
}
