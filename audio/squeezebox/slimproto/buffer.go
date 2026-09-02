package slimproto

import (
	"errors"
	"io"
	"sync"
)

var (
	ErrBufferClosed = errors.New("audio buffer closed")
)

// AudioRingBuffer provides a thread-safe circular byte buffer with backpressure.
// Writers block when the buffer does not have enough capacity, and are unblocked
// as readers consume bytes or when the buffer is flushed/closed.
type AudioRingBuffer struct {
	buf      []byte
	size     int
	readPos  int
	writePos int
	count    int

	mu     sync.Mutex
	cond   *sync.Cond
	closed bool
}

// NewAudioRingBuffer creates an initialized AudioRingBuffer with the given capacity in bytes.
func NewAudioRingBuffer(size int) *AudioRingBuffer {
	if size <= 0 {
		size = 2 * 1024 * 1024 // 2 MB default
	}
	rb := &AudioRingBuffer{
		buf:  make([]byte, size),
		size: size,
	}
	rb.cond = sync.NewCond(&rb.mu)
	return rb
}

// Write writes data into the ring buffer. If the buffer is full, Write blocks
// until enough space is freed by readers, or until the buffer is flushed/closed.
func (rb *AudioRingBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	totalWritten := 0
	for totalWritten < len(p) {
		for (rb.size - rb.count) == 0 {
			if rb.closed {
				return totalWritten, ErrBufferClosed
			}
			rb.cond.Wait()
		}

		if rb.closed {
			return totalWritten, ErrBufferClosed
		}

		availableSpace := rb.size - rb.count
		chunk := min(len(p)-totalWritten, availableSpace)

		// First segment up to end of underlying slice
		firstPart := min(chunk, rb.size-rb.writePos)
		copy(rb.buf[rb.writePos:rb.writePos+firstPart], p[totalWritten:totalWritten+firstPart])

		// Second wrapped segment from beginning of underlying slice
		secondPart := chunk - firstPart
		if secondPart > 0 {
			copy(rb.buf[0:secondPart], p[totalWritten+firstPart:totalWritten+chunk])
		}

		rb.writePos = (rb.writePos + chunk) % rb.size
		rb.count += chunk
		totalWritten += chunk

		rb.cond.Broadcast()
	}

	return totalWritten, nil
}

// Read reads up to len(p) bytes from the ring buffer non-blockingly (reads available bytes).
func (rb *AudioRingBuffer) Read(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		if rb.closed {
			return 0, io.EOF
		}
		return 0, nil
	}

	chunk := min(len(p), rb.count)

	firstPart := min(chunk, rb.size-rb.readPos)
	copy(p[0:firstPart], rb.buf[rb.readPos:rb.readPos+firstPart])

	secondPart := chunk - firstPart
	if secondPart > 0 {
		copy(p[firstPart:chunk], rb.buf[0:secondPart])
	}

	rb.readPos = (rb.readPos + chunk) % rb.size
	rb.count -= chunk

	rb.cond.Broadcast()
	return chunk, nil
}

// Available returns the number of bytes currently stored and ready to read.
func (rb *AudioRingBuffer) Available() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}

// Space returns the remaining free byte capacity in the buffer.
func (rb *AudioRingBuffer) Space() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.size - rb.count
}

// Capacity returns the total capacity of the buffer in bytes.
func (rb *AudioRingBuffer) Capacity() int {
	return rb.size
}

// Flush discards all unread bytes in the buffer and wakes any blocked writers.
func (rb *AudioRingBuffer) Flush() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.readPos = 0
	rb.writePos = 0
	rb.count = 0
	rb.cond.Broadcast()
}

// Close marks the buffer as closed and wakes all waiting goroutines.
func (rb *AudioRingBuffer) Close() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.closed = true
	rb.cond.Broadcast()
}
