package slimproto

import (
	"time"

	"golang.org/x/sys/unix"
)

// Clock provides monotonic time in milliseconds for SlimProto jiffies calculations
// and real-time audio consumption pacing.
type Clock interface {
	// NowMonotonicMs returns the system monotonic clock in milliseconds.
	NowMonotonicMs() uint32
	// Now returns the current wall clock time.
	Now() time.Time
}

// SystemClock implements Clock using the operating system's monotonic clock.
type SystemClock struct{}

// NewSystemClock creates a new default SystemClock.
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// NowMonotonicMs returns the monotonic clock in milliseconds matching Squeezelite gettime_ms().
func (s *SystemClock) NowMonotonicMs() uint32 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err == nil {
		return uint32(ts.Sec*1000 + ts.Nsec/1000000)
	}
	return uint32(time.Now().UnixMilli())
}

// Now returns the current system time.
func (s *SystemClock) Now() time.Time {
	return time.Now()
}
