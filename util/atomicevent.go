package util

import (
	"sync"
)

// AtomicEvent holds a single, latest event and provides non-blocking updates.
// Only the most recent event is retained.
type AtomicEvent[T any] struct {
	mu     sync.Mutex
	val    T
	hasVal bool
	notify chan struct{} // Buffered channel of size 1 for notification
}

// NewAtomicEvent creates a new AtomicEvent instance.
func NewAtomicEvent[T any]() *AtomicEvent[T] {
	return &AtomicEvent[T]{
		notify: make(chan struct{}, 1), // Buffered channel with capacity 1
	}
}

// Send updates with the latest event. It is non-blocking.
// If an unconsumed event was displaced, it returns that event and true.
func (ae *AtomicEvent[T]) Send(event T) (displaced T, hadPrevious bool) {
	ae.mu.Lock()
	if ae.hasVal {
		displaced = ae.val
		hadPrevious = true
	}
	ae.val = event
	ae.hasVal = true
	ae.mu.Unlock()

	select {
	case ae.notify <- struct{}{}:
		// Notification sent successfully.
	default:
		// Channel was already full, notification is already pending.
	}

	return displaced, hadPrevious
}

// Channel returns the notification channel for use in select statements.
func (ae *AtomicEvent[T]) Channel() <-chan struct{} {
	return ae.notify
}

// Consume retrieves and consumes the current latest event, marking the
// event as empty. Returns false if no event is present.
func (ae *AtomicEvent[T]) Consume() (T, bool) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if !ae.hasVal {
		var zero T
		return zero, false
	}

	val := ae.val
	var zero T
	ae.val = zero
	ae.hasVal = false

	return val, true
}

// HasPending checks if a notification is waiting to be consumed.
// This is a non-destructive check on the notification channel.
func (ae *AtomicEvent[T]) HasPending() bool {
	return len(ae.notify) > 0
}

// AtomicMapEvent holds a map of events, allowing for non-blocking updates
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

// ConsumeValuesInto copies the pending events into the provided dst map,
// then clears the internal map. The internal map is never exposed.
func (ae *AtomicMapEvent[T]) ConsumeValuesInto(dst map[string]T) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	clear(dst)
	for k, v := range ae.value {
		dst[k] = v
	}
	clear(ae.value)
}

// HasPending checks if a notification is waiting to be consumed.
// This is a non-destructive check.
func (ae *AtomicMapEvent[T]) HasPending() bool {
	return len(ae.notify) > 0
}
