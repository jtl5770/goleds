package audio

import (
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

// LevelsSnapshot captures an immutable stereo audio measurement.
type LevelsSnapshot struct {
	LeftDB  float64
	RightDB float64
	Active  bool
}

// AtomicLevels stores instantaneous stereo audio levels using atomic pointer snapshots,
// guaranteeing a 100% atomic read snapshot with zero allocations on the read path.
type AtomicLevels struct {
	snapshot atomic.Pointer[LevelsSnapshot]
}

// NewAtomicLevels creates an initialized AtomicLevels instance with silence (-100 dB).
func NewAtomicLevels() *AtomicLevels {
	al := &AtomicLevels{}
	al.snapshot.Store(&LevelsSnapshot{
		LeftDB:  -100,
		RightDB: -100,
		Active:  false,
	})
	return al
}

// Set stores the dB levels and active state in an atomic transaction.
func (a *AtomicLevels) Set(leftDB, rightDB float64, active bool) {
	a.snapshot.Store(&LevelsSnapshot{
		LeftDB:  leftDB,
		RightDB: rightDB,
		Active:  active,
	})
}

// Get loads the current dB levels and active state atomically with zero allocations.
func (a *AtomicLevels) Get() (leftDB, rightDB float64, active bool) {
	s := a.snapshot.Load()
	if s == nil {
		return -100, -100, false
	}
	return s.LeftDB, s.RightDB, s.Active
}
