package slimproto

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

func TestAudioRingBuffer_BasicReadWrite(t *testing.T) {
	rb := NewAudioRingBuffer(100)

	if rb.Capacity() != 100 {
		t.Fatalf("Expected capacity 100, got %d", rb.Capacity())
	}
	if rb.Available() != 0 {
		t.Fatalf("Expected available 0, got %d", rb.Available())
	}
	if rb.Space() != 100 {
		t.Fatalf("Expected space 100, got %d", rb.Space())
	}

	data := []byte("Hello, Squeezebox SlimProto Audio Buffer!")
	n, err := rb.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Expected written %d, got %d", len(data), n)
	}
	if rb.Available() != len(data) {
		t.Fatalf("Expected available %d, got %d", len(data), rb.Available())
	}

	readBuf := make([]byte, len(data))
	rn, err := rb.Read(readBuf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if rn != len(data) {
		t.Fatalf("Expected read %d, got %d", len(data), rn)
	}
	if !bytes.Equal(readBuf, data) {
		t.Fatalf("Read mismatch: expected %q, got %q", string(data), string(readBuf))
	}
	if rb.Available() != 0 {
		t.Fatalf("Expected available 0 after read, got %d", rb.Available())
	}
}

func TestAudioRingBuffer_WrapAround(t *testing.T) {
	rb := NewAudioRingBuffer(10)

	// Write 8 bytes
	_, _ = rb.Write([]byte("12345678"))
	// Read 6 bytes -> readPos is at 6
	r := make([]byte, 6)
	_, _ = rb.Read(r)
	if string(r) != "123456" {
		t.Fatalf("Expected 123456, got %s", string(r))
	}

	// Write 6 bytes -> should wrap around past index 10 to 0..4
	_, _ = rb.Write([]byte("abcdef"))
	if rb.Available() != 8 {
		t.Fatalf("Expected available 8, got %d", rb.Available())
	}

	// Read all 8 bytes
	r2 := make([]byte, 8)
	n, _ := rb.Read(r2)
	if n != 8 {
		t.Fatalf("Expected read 8, got %d", n)
	}
	if string(r2) != "78abcdef" {
		t.Fatalf("Expected 78abcdef, got %s", string(r2))
	}
}

func TestAudioRingBuffer_Flush(t *testing.T) {
	rb := NewAudioRingBuffer(50)
	_, _ = rb.Write([]byte("Test data before flush"))
	if rb.Available() == 0 {
		t.Fatal("Expected buffer to contain data before flush")
	}

	rb.Flush()
	if rb.Available() != 0 {
		t.Fatalf("Expected available 0 after flush, got %d", rb.Available())
	}

	// Should be able to write again immediately after flush
	_, err := rb.Write([]byte("New data"))
	if err != nil {
		t.Fatalf("Write after flush failed: %v", err)
	}
	if rb.Available() != len("New data") {
		t.Fatalf("Expected available %d, got %d", len("New data"), rb.Available())
	}
}

func TestAudioRingBuffer_ConcurrentReadWrite(t *testing.T) {
	rb := NewAudioRingBuffer(1024)
	totalBytes := 100000

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		chunk := make([]byte, 256)
		for i := range chunk {
			chunk[i] = byte(i % 256)
		}
		written := 0
		for written < totalBytes {
			toWrite := min(len(chunk), totalBytes-written)
			n, err := rb.Write(chunk[:toWrite])
			if err != nil {
				return
			}
			written += n
		}
	}()

	var readBytes []byte
	go func() {
		defer wg.Done()
		buf := make([]byte, 128)
		for len(readBytes) < totalBytes {
			n, err := rb.Read(buf)
			if err == io.EOF {
				break
			}
			if n > 0 {
				readBytes = append(readBytes, buf[:n]...)
			} else {
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	wg.Wait()

	if len(readBytes) != totalBytes {
		t.Fatalf("Expected %d bytes, got %d", totalBytes, len(readBytes))
	}
	for i, b := range readBytes {
		if b != byte(i%256) {
			t.Fatalf("Mismatch at byte %d: expected %d, got %d", i, byte(i%256), b)
		}
	}
}
