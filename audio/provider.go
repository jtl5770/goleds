package audio

import (
	"math"
	"sync/atomic"
)

// AudioProvider provides thread-safe, lock-free audio level measurements.
type AudioProvider interface {
	// GetLevels returns the latest left and right dB levels and whether audio is active.
	GetLevels() (leftDB, rightDB float64, active bool)
	// Start starts the audio provider background worker.
	Start() error
	// Stop stops the audio provider and gracefully cleans up resources.
	Stop() error
}

// AtomicLevels stores instantaneous stereo audio levels using atomic primitives,
// guaranteeing zero heap allocations on both read and write paths.
type AtomicLevels struct {
	leftBits  atomic.Uint64
	rightBits atomic.Uint64
	active    atomic.Bool
}

// NewAtomicLevels creates an initialized AtomicLevels instance with silence (-100 dB).
func NewAtomicLevels() *AtomicLevels {
	al := &AtomicLevels{}
	al.Set(-100, -100, false)
	return al
}

// Set stores the dB levels and active state atomically.
func (a *AtomicLevels) Set(leftDB, rightDB float64, active bool) {
	a.leftBits.Store(math.Float64bits(leftDB))
	a.rightBits.Store(math.Float64bits(rightDB))
	a.active.Store(active)
}

// Get loads the current dB levels and active state atomically with zero allocations.
func (a *AtomicLevels) Get() (leftDB, rightDB float64, active bool) {
	return math.Float64frombits(a.leftBits.Load()),
		math.Float64frombits(a.rightBits.Load()),
		a.active.Load()
}
