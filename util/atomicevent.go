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

// AtomicSetEvent holds a map of events, allowing for non-blocking updates

type AtomicMapEvent[T any] struct {
	mu     sync.Mutex    // Protects access to 'value'
	value  map[string]T  // The latest event
	notify chan struct{} // Buffered channel of size 1 for notification
}

// NewAtomicMapEvent creates a new AtomicMapEvent instance.
func NewAtomicMapEvent[T any]() *AtomicMapEvent[T] {
	return &AtomicMapEvent[T]{
		notify: make(chan struct{}, 1),
		value:  make(map[string]T),
	}
}

// Send updates with the latest event for a matching. It is non-blocking.
func (ae *AtomicMapEvent[T]) Send(key string, event T) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ae.value[key] = event

	select {
	case ae.notify <- struct{}{}:
		// Notification sent successfully.
	default:
		// Channel was already full, notification is already pending.
	}
}

// Channel returns the notification channel for use in select statements.
func (ae *AtomicMapEvent[T]) Channel() <-chan struct{} {
	return ae.notify
}

// Value returns the current latest event.
func (ae *AtomicMapEvent[T]) Value() map[string]T {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ret := make(map[string]T, len(ae.value))
	for key, value := range ae.value {
		ret[key] = value
	}
	return ret
}

// HasPending checks if a notification is waiting to be consumed.
// This is a non-destructive check.
func (ae *AtomicMapEvent[T]) HasPending() bool {
	return len(ae.notify) > 0
}
