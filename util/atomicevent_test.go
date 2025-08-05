package util

import (
	"fmt"
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

func TestNewAtomicMapEvent(t *testing.T) {
	ae := NewAtomicMapEvent[any]()
	assert.NotNil(t, ae, "NewAtomicMapEvent should not return nil")
	assert.NotNil(t, ae.notify, "notify channel should be initialized")
	assert.NotNil(t, ae.value, "value map should be initialized")
}

func TestAtomicMapEvent_SendAndConsumeValues(t *testing.T) {
	ae := NewAtomicMapEvent[int]()

	ae.Send("one", 1)
	ae.Send("two", 2)

	assert.True(t, ae.HasPending(), "should have pending notification")
	select {
	case <-ae.Channel():
		// Good, notification received
	default:
		t.Fatal("should have received a notification")
	}

	values := ae.ConsumeValues()
	assert.Len(t, values, 2, "should have two values")
	assert.Equal(t, 1, values["one"])
	assert.Equal(t, 2, values["two"])

	// After consuming, the internal map should be empty and no notification should be pending
	// (unless another Send happens)
	assert.False(t, ae.HasPending(), "should not have pending notification after consume")
	select {
	case <-ae.Channel():
		t.Fatal("channel should be empty after consume")
	default:
		// Good
	}

	// Consuming again should yield an empty map
	values = ae.ConsumeValues()
	assert.Len(t, values, 0, "should have zero values after consuming")

	// Send again to ensure it still works
	ae.Send("three", 3)
	values = ae.ConsumeValues()
	assert.Len(t, values, 1)
	assert.Equal(t, 3, values["three"])
}

func TestAtomicMapEvent_SendOverwrites(t *testing.T) {
	ae := NewAtomicMapEvent[string]()
	ae.Send("key1", "initial")
	ae.Send("key1", "overwrite")

	values := ae.ConsumeValues()
	assert.Len(t, values, 1)
	assert.Equal(t, "overwrite", values["key1"])
}

func TestAtomicMapEvent_Concurrency(t *testing.T) {
	ae := NewAtomicMapEvent[int]()
	var wg sync.WaitGroup
	const numGoroutines = 10
	const numWritesPerGoRoutine = 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numWritesPerGoRoutine; j++ {
				key := fmt.Sprintf("g%d-k%d", goroutineID, j)
				ae.Send(key, j)
			}
		}(i)
	}

	wg.Wait()

	// Wait for notification
	<-ae.Channel()

	values := ae.ConsumeValues()
	assert.Len(t, values, numGoroutines*numWritesPerGoRoutine)

	// After consuming, should be empty
	assert.Len(t, ae.ConsumeValues(), 0)
}

func TestAtomicMapEvent_ConcurrentReadWrite(t *testing.T) {
	ae := NewAtomicMapEvent[int]()
	var writerWg sync.WaitGroup
	var consumerWg sync.WaitGroup
	const numWriters = 10
	const numWritesPerWriter = 100
	totalWrites := numWriters * numWritesPerWriter

	consumedValues := make(map[string]int)
	var consumedMutex sync.Mutex

	// Helper to process a map of consumed values safely from multiple goroutines
	processConsumed := func(consumed map[string]int) {
		if len(consumed) == 0 {
			return
		}
		consumedMutex.Lock()
		defer consumedMutex.Unlock()
		for k, v := range consumed {
			if _, exists := consumedValues[k]; exists {
				t.Errorf("key %s was consumed twice", k)
			}
			consumedValues[k] = v
		}
	}

	consumerDone := make(chan struct{})

	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		for {
			select {
			case <-ae.Channel():
				processConsumed(ae.ConsumeValues())
			case <-consumerDone:
				// Writers are done. Perform a final drain.
			finalDrainLoop:
				for {
					select {
					case <-ae.Channel():
						processConsumed(ae.ConsumeValues())
					default:
						// No more notifications.
						break finalDrainLoop
					}
				}
				// One final check to be absolutely sure no values were missed.
				processConsumed(ae.ConsumeValues())
				return
			}
		}
	}()

	writerWg.Add(numWriters)
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer writerWg.Done()
			for j := 0; j < numWritesPerWriter; j++ {
				key := fmt.Sprintf("w%d-k%d", writerID, j)
				ae.Send(key, j)
			}
		}(i)
	}

	writerWg.Wait()     // Wait for all writers to finish
	close(consumerDone) // Signal consumer to stop
	consumerWg.Wait()   // Wait for consumer to finish processing

	consumedMutex.Lock()
	defer consumedMutex.Unlock()
	assert.Len(t, consumedValues, totalWrites, "all written values should have been consumed exactly once")
}
