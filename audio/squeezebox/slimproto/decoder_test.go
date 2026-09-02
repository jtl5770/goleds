package slimproto

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mewkiz/flac/frame"
)

type mockDecoderCallback struct {
	mu               sync.Mutex
	sampleRate       uint32
	channels         uint32
	thresholdReached bool
}

func (m *mockDecoderCallback) OnSampleRateChanged(sr uint32, ch uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sampleRate = sr
	m.channels = ch
}

func (m *mockDecoderCallback) OnThresholdReached() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.thresholdReached = true
}

func TestPCMDecoder_BasicStream(t *testing.T) {
	dec := NewPCMDecoder()
	rb := NewAudioRingBuffer(65536)
	cb := &mockDecoderCallback{}

	// Generate 1000 stereo 16-bit frames (4000 bytes)
	data := make([]byte, 4000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	reader := bytes.NewReader(data)
	ctx := context.Background()

	err := dec.Decode(ctx, reader, rb, 2000, cb)
	if err != nil {
		t.Fatalf("Unexpected error decoding PCM: %v", err)
	}

	if rb.Available() != 4000 {
		t.Errorf("Expected 4000 bytes in ring buffer, got %d", rb.Available())
	}

	cb.mu.Lock()
	if !cb.thresholdReached {
		t.Errorf("Expected thresholdReached to be true")
	}
	cb.mu.Unlock()
}

func TestPCMDecoder_ContextCancel(t *testing.T) {
	dec := NewPCMDecoder()
	rb := NewAudioRingBuffer(1024)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	reader := bytes.NewReader(make([]byte, 2048))
	err := dec.Decode(ctx, reader, rb, 512, nil)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestFLACDecoder_ProcessFrame(t *testing.T) {
	dec := NewFLACDecoder()
	rb := NewAudioRingBuffer(65536)
	cb := &mockDecoderCallback{}

	// Build a mock FLAC frame manually
	f := &frame.Frame{
		Header: frame.Header{
			SampleRate:    48000,
			BitsPerSample: 16,
			BlockSize:     64,
		},
		Subframes: []*frame.Subframe{
			{Samples: make([]int32, 64)},
			{Samples: make([]int32, 64)},
		},
	}

	for i := 0; i < 64; i++ {
		f.Subframes[0].Samples[i] = int32(i * 100)
		f.Subframes[1].Samples[i] = int32(-i * 100)
	}

	sentThreshold := false
	err := dec.processFrame(f, rb, 128, &sentThreshold, cb)
	if err != nil {
		t.Fatalf("processFrame failed: %v", err)
	}

	// 64 samples * 2 channels * 2 bytes = 256 bytes
	if rb.Available() != 256 {
		t.Errorf("Expected 256 bytes in ring buffer, got %d", rb.Available())
	}

	cb.mu.Lock()
	if cb.sampleRate != 48000 || cb.channels != 2 {
		t.Errorf("Expected (48000, 2), got (%d, %d)", cb.sampleRate, cb.channels)
	}
	if !cb.thresholdReached {
		t.Errorf("Expected threshold reached to be true")
	}
	cb.mu.Unlock()
}

func TestFLACDecoder_BitDepthScaling(t *testing.T) {
	dec := NewFLACDecoder()
	rb := NewAudioRingBuffer(65536)

	// 24-bit audio frame
	f24 := &frame.Frame{
		Header: frame.Header{
			SampleRate:    44100,
			BitsPerSample: 24,
			BlockSize:     1,
		},
		Subframes: []*frame.Subframe{
			{Samples: []int32{0x007FFF00}}, // Should downshift to ~0x7FFF
			{Samples: []int32{-0x007FFF00}},
		},
	}

	sentThreshold := false
	err := dec.processFrame(f24, rb, 4, &sentThreshold, nil)
	if err != nil {
		t.Fatalf("processFrame 24-bit failed: %v", err)
	}

	out := make([]byte, 4)
	n, _ := rb.Read(out)
	if n != 4 {
		t.Fatalf("Expected 4 bytes read, got %d", n)
	}
}

func TestStreamFormat_String(t *testing.T) {
	tests := []struct {
		format   StreamFormat
		expected string
	}{
		{FormatFLAC, "FLAC"},
		{FormatPCM, "PCM"},
		{FormatMP3, "MP3"},
		{FormatAAC, "AAC"},
		{FormatOGG, "OGG"},
		{StreamFormat('x'), "x"},
	}

	for _, tt := range tests {
		if tt.format.String() != tt.expected {
			t.Errorf("Expected %s for format %c, got %s", tt.expected, tt.format, tt.format.String())
		}
	}
}

func TestFLACDecoder_TimeoutAndThreshold(t *testing.T) {
	dec := NewFLACDecoder()
	rb := NewAudioRingBuffer(1024)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	// Empty stream (will hit EOF immediately)
	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(ctx, reader, rb, 100, nil)
}
