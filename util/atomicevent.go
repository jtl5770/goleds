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
// If a notification is already pending (i.e., the 'notify' channel is full),
// it will simply overwrite the internal value without sending another notification.
func (ae *AtomicEvent[T]) Send(event T) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	ae.value = event // Always update the latest value

	select {
	case ae.notify <- struct{}{}:
		// Notification sent successfully. This happens if the 'notify' channel
		// was empty (no pending notification).
	default:
		// The 'notify' channel was already full, meaning a notification
		// is already pending. No need to send another one; the receiver
		// will still get the latest value when it processes the pending notification.
	}
}

// Channel returns the notification channel.
// This allows the caller to use it in a select statement.
// After receiving from this channel, the caller should call Value() to get the latest event.
func (ae *AtomicEvent[T]) Channel() <-chan struct{} {
	return ae.notify
}

// Value returns the current latest event.
// This method is safe for concurrent access.
func (ae *AtomicEvent[T]) Value() T {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	return ae.value
}
