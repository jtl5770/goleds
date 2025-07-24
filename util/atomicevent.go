package util

import (
	"sync"
)

// AtomicEvent holds a single, latest event and provides non-blocking updates.
// Only the most recent event is retained.
type AtomicEvent[T any] struct {
	mu     sync.Mutex    // Protects access to 'value'
	value  T             // The latest event
	notify chan struct{} // Buffered channel of size 1 for notification
}

// NewAtomicEvent creates a new AtomicEvent instance.
func NewAtomicEvent[T any]() *AtomicEvent[T] {
	return &AtomicEvent[T]{
		notify: make(chan struct{}, 1), // Buffered channel with capacity 1
	}
}

// Send updates with the latest event. It is non-blocking.
func (ae *AtomicEvent[T]) Send(event T) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ae.value = event // Always update the latest value

	select {
	case ae.notify <- struct{}{}:
		// Notification sent successfully.
	default:
		// Channel was already full, notification is already pending.
	}
}

// Channel returns the notification channel for use in select statements.
func (ae *AtomicEvent[T]) Channel() <-chan struct{} {
	return ae.notify
}

// Value returns the current latest event.
func (ae *AtomicEvent[T]) Value() T {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	return ae.value
}

// HasPending checks if a notification is waiting to be consumed.
// This is a non-destructive check.
func (ae *AtomicEvent[T]) HasPending() bool {
	return len(ae.notify) > 0
}