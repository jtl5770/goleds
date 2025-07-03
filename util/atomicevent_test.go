package util

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAtomicEvent(t *testing.T) {
	ae := NewAtomicEvent[any]()
	assert.NotNil(t, ae, "NewAtomicEvent should not return nil")
	assert.NotNil(t, ae.notify, "notify channel should be initialized")
}

func TestSendAndValue(t *testing.T) {
	// Test with an integer
	aeInt := NewAtomicEvent[int]()
	aeInt.Send(123)
	assert.Equal(t, 123, aeInt.Value(), "Value should be 123")

	// Test with a string
	aeStr := NewAtomicEvent[string]()
	aeStr.Send("hello")
	assert.Equal(t, "hello", aeStr.Value(), "Value should be 'hello'")

	// Test with a struct
	type testStruct struct {
		Field int
	}
	ts := testStruct{Field: 42}
	aeStruct := NewAtomicEvent[testStruct]()
	aeStruct.Send(ts)
	assert.Equal(t, ts, aeStruct.Value(), "Value should be the test struct")
}

func TestNotificationChannel(t *testing.T) {
	ae := NewAtomicEvent[string]()

	// Send an event, should get a notification
	ae.Send("event1")
	select {
	case <-ae.Channel():
		// Good, got notification
	default:
		t.Fatal("should have received a notification")
	}

	// The channel should be empty now
	select {
	case <-ae.Channel():
		t.Fatal("channel should be empty")
	default:
		// Good, channel is empty
	}

	// Send multiple events, should only get one notification
	ae.Send("event2")
	ae.Send("event3")
	select {
	case <-ae.Channel():
		// Good, got notification
	default:
		t.Fatal("should have received a notification")
	}

	// The channel should be empty now
	select {
	case <-ae.Channel():
		t.Fatal("channel should be empty")
	default:
		// Good, channel is empty
	}

	// Check the value is the latest one
	assert.Equal(t, "event3", ae.Value(), "Value should be the last event sent")
}

func TestConcurrency(t *testing.T) {
	ae := NewAtomicEvent[int]()
	done := make(chan struct{})

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			ae.Send(i)
		}
		close(done)
	}()

	// Reader goroutine
	lastRead := -1
	var readerWg sync.WaitGroup
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for {
			select {
			case <-ae.Channel():
				val := ae.Value()
				if val < lastRead {
					t.Errorf("read a stale value: got %d, last was %d", val, lastRead)
				}
				lastRead = val
			case <-done:
				// Drain the channel one last time to avoid a race.
				select {
				case <-ae.Channel():
					val := ae.Value()
					if val < lastRead {
						t.Errorf("read a stale value: got %d, last was %d", val, lastRead)
					}
					lastRead = val
				default:
				}
				return
			}
		}
	}()

	readerWg.Wait()

	assert.Equal(t, 999, ae.Value(), "Final value should be 999")
}
