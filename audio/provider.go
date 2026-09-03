package audio

import "github.com/jtl5770/go-slimvu"

// AudioProvider provides thread-safe, lock-free audio level measurements.
type AudioProvider = slimvu.AudioProvider

// LevelsSnapshot captures an immutable stereo audio measurement.
type LevelsSnapshot = slimvu.LevelsSnapshot

// AtomicLevels stores instantaneous stereo audio levels using atomic pointer snapshots,
// guaranteeing a 100% atomic read snapshot with zero allocations on the read path.
type AtomicLevels = slimvu.AtomicLevels

// NewAtomicLevels creates an initialized AtomicLevels instance with silence (-100 dB).
func NewAtomicLevels() *AtomicLevels {
	return slimvu.NewAtomicLevels()
}
