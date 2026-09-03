package slimproto

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mewkiz/flac/frame"
	"lautenbacher.net/goleds/audio/dsp"
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
	dec := NewPCMDecoder(PCMConfig{
		SampleRate:    44100,
		BitsPerSample: 16,
		Channels:      2,
		LittleEndian:  true,
	})
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
	if cb.sampleRate != 44100 || cb.channels != 2 {
		t.Errorf("Expected (44100, 2), got (%d, %d)", cb.sampleRate, cb.channels)
	}
	cb.mu.Unlock()
}

func TestPCMDecoder_BigEndian16BitStereo(t *testing.T) {
	dec := NewPCMDecoder(PCMConfig{
		SampleRate:    48000,
		BitsPerSample: 16,
		Channels:      2,
		LittleEndian:  false, // Big-Endian from LMS
	})
	rb := NewAudioRingBuffer(1024)

	// Sample: Left = 0x1234 (4660), Right = -0x1234 (-4660 = 0xEDCC)
	inData := []byte{0x12, 0x34, 0xED, 0xCC}
	reader := bytes.NewReader(inData)

	err := dec.Decode(context.Background(), reader, rb, 4, nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	out := make([]byte, 4)
	n, _ := rb.Read(out)
	if n != 4 {
		t.Fatalf("Expected 4 bytes read, got %d", n)
	}

	left := int16(binary.LittleEndian.Uint16(out[0:2]))
	right := int16(binary.LittleEndian.Uint16(out[2:4]))

	if left != 0x1234 {
		t.Errorf("Expected left 0x1234, got 0x%X", left)
	}
	if right != -0x1234 {
		t.Errorf("Expected right -0x1234, got 0x%X", right)
	}
}

func TestPCMDecoder_MonoDuplicateToStereo(t *testing.T) {
	dec := NewPCMDecoder(PCMConfig{
		SampleRate:    22050,
		BitsPerSample: 16,
		Channels:      1, // Mono
		LittleEndian:  true,
	})
	rb := NewAudioRingBuffer(1024)

	inData := []byte{0x56, 0x78} // Single 16-bit mono sample
	reader := bytes.NewReader(inData)

	err := dec.Decode(context.Background(), reader, rb, 4, nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	out := make([]byte, 4)
	n, _ := rb.Read(out)
	if n != 4 {
		t.Fatalf("Expected 4 bytes read (stereo duplicated), got %d", n)
	}

	left := int16(binary.LittleEndian.Uint16(out[0:2]))
	right := int16(binary.LittleEndian.Uint16(out[2:4]))

	if left != 0x7856 || right != 0x7856 {
		t.Errorf("Expected mono duplicated to stereo (0x7856, 0x7856), got (%x, %x)", left, right)
	}
}

func TestPCMDecoder_24BitScaling(t *testing.T) {
	dec := NewPCMDecoder(PCMConfig{
		SampleRate:    96000,
		BitsPerSample: 24,
		Channels:      2,
		LittleEndian:  false, // Big-Endian
	})
	rb := NewAudioRingBuffer(1024)

	// 24-bit sample: Left = 0x7FFF00 (downscales to 0x7FFF), Right = 0x800000 (downscales to -0x8000)
	inData := []byte{0x7F, 0xFF, 0x00, 0x80, 0x00, 0x00}
	reader := bytes.NewReader(inData)

	err := dec.Decode(context.Background(), reader, rb, 4, nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	out := make([]byte, 4)
	n, _ := rb.Read(out)
	if n != 4 {
		t.Fatalf("Expected 4 bytes read, got %d", n)
	}

	left := int16(binary.LittleEndian.Uint16(out[0:2]))
	right := int16(binary.LittleEndian.Uint16(out[2:4]))

	if left != 0x7FFF {
		t.Errorf("Expected left 0x7FFF, got 0x%X", left)
	}
	if right != -32768 {
		t.Errorf("Expected right -32768, got %d", right)
	}
}

func TestPCMDecoder_8BitUnsigned(t *testing.T) {
	dec := NewPCMDecoder(PCMConfig{
		SampleRate:    11025,
		BitsPerSample: 8,
		Channels:      2,
		LittleEndian:  true,
	})
	rb := NewAudioRingBuffer(1024)

	// 8-bit unsigned: 128 = 0 (silence), 255 = ~32512 (max pos), 0 = -32768 (max neg)
	inData := []byte{128, 255}
	reader := bytes.NewReader(inData)

	err := dec.Decode(context.Background(), reader, rb, 4, nil)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	out := make([]byte, 4)
	n, _ := rb.Read(out)
	if n != 4 {
		t.Fatalf("Expected 4 bytes read, got %d", n)
	}

	left := int16(binary.LittleEndian.Uint16(out[0:2]))
	right := int16(binary.LittleEndian.Uint16(out[2:4]))

	if left != 0 {
		t.Errorf("Expected left 0 for midpoint 128, got %d", left)
	}
	if right != (127 << 8) {
		t.Errorf("Expected right %d for 255, got %d", 127<<8, right)
	}
}

func TestParsePCMConfig(t *testing.T) {
	cfg := ParsePCMConfig('4', '2', '1', '1')
	if cfg.SampleRate != 48000 {
		t.Errorf("Expected 48000, got %d", cfg.SampleRate)
	}
	if cfg.BitsPerSample != 24 {
		t.Errorf("Expected 24 bits, got %d", cfg.BitsPerSample)
	}
	if cfg.Channels != 1 {
		t.Errorf("Expected 1 channel, got %d", cfg.Channels)
	}
	if !cfg.LittleEndian {
		t.Errorf("Expected LittleEndian true")
	}

	cfgDefault := ParsePCMConfig('x', 'x', 'x', 'x')
	if cfgDefault.SampleRate != 44100 || cfgDefault.BitsPerSample != 16 || cfgDefault.Channels != 2 || cfgDefault.LittleEndian {
		t.Errorf("Unexpected default fallback config: %+v", cfgDefault)
	}
}

func TestPCMDecoder_ContextCancel(t *testing.T) {
	dec := NewPCMDecoder(PCMConfig{SampleRate: 44100, BitsPerSample: 16, Channels: 2, LittleEndian: true})
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

func TestMP3Decoder_InvalidStream(t *testing.T) {
	dec := NewMP3Decoder()
	rb := NewAudioRingBuffer(1024)

	reader := bytes.NewReader([]byte{0x00, 0x01, 0x02})
	err := dec.Decode(context.Background(), reader, rb, 100, nil)
	if err == nil {
		t.Errorf("Expected error decoding invalid MP3 data, got nil")
	}
}

func TestMP3Decoder_ContextCancel(t *testing.T) {
	dec := NewMP3Decoder()
	rb := NewAudioRingBuffer(1024)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(ctx, reader, rb, 100, nil)
}

func TestAACDecoder_EmptyStream(t *testing.T) {
	dec := NewAACDecoder()
	rb := NewAudioRingBuffer(1024)

	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(context.Background(), reader, rb, 100, nil)
}

func TestAACDecoder_ContextCancel(t *testing.T) {
	dec := NewAACDecoder()
	rb := NewAudioRingBuffer(1024)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(ctx, reader, rb, 100, nil)
}

func TestFLACDecoder_SyntheticFile(t *testing.T) {
	data, err := os.ReadFile("testdata/sine.flac")
	if err != nil {
		t.Fatalf("Failed to read test vector sine.flac: %v", err)
	}

	dec := NewFLACDecoder()
	rb := NewAudioRingBuffer(128 * 1024)
	cb := &mockDecoderCallback{}

	err = dec.Decode(context.Background(), bytes.NewReader(data), rb, 512, cb)
	if err != nil {
		t.Fatalf("FLACDecoder failed to decode synthetic sine.flac: %v", err)
	}

	if rb.Available() == 0 {
		t.Fatalf("Expected decoded PCM samples in ring buffer, got 0 bytes")
	}

	cb.mu.Lock()
	if cb.sampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", cb.sampleRate)
	}
	if !cb.thresholdReached {
		t.Errorf("Expected thresholdReached to be true")
	}
	cb.mu.Unlock()

	// Verify that audio level calculation produces valid active signal
	pcm := make([]byte, rb.Available())
	_, _ = rb.Read(pcm)
	leftDB, rightDB := dsp.CalculateLevels(pcm)
	if leftDB <= -90 || rightDB <= -90 {
		t.Errorf("Expected active audio levels from sine.flac, got LeftRMS: %f, RightRMS: %f", leftDB, rightDB)
	}
}

func TestMP3Decoder_SyntheticFile(t *testing.T) {
	data, err := os.ReadFile("testdata/sine.mp3")
	if err != nil {
		t.Fatalf("Failed to read test vector sine.mp3: %v", err)
	}

	dec := NewMP3Decoder()
	rb := NewAudioRingBuffer(128 * 1024)
	cb := &mockDecoderCallback{}

	err = dec.Decode(context.Background(), bytes.NewReader(data), rb, 512, cb)
	if err != nil {
		t.Fatalf("MP3Decoder failed to decode synthetic sine.mp3: %v", err)
	}

	if rb.Available() == 0 {
		t.Fatalf("Expected decoded PCM samples in ring buffer, got 0 bytes")
	}

	cb.mu.Lock()
	if cb.sampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", cb.sampleRate)
	}
	if !cb.thresholdReached {
		t.Errorf("Expected thresholdReached to be true")
	}
	cb.mu.Unlock()

	// Verify that audio level calculation produces valid active signal
	pcm := make([]byte, rb.Available())
	_, _ = rb.Read(pcm)
	leftDB, rightDB := dsp.CalculateLevels(pcm)
	if leftDB <= -90 || rightDB <= -90 {
		t.Errorf("Expected active audio levels from sine.mp3, got LeftRMS: %f, RightRMS: %f", leftDB, rightDB)
	}
}

func TestAACDecoder_SyntheticFile(t *testing.T) {
	data, err := os.ReadFile("testdata/sine.aac")
	if err != nil {
		t.Fatalf("Failed to read test vector sine.aac: %v", err)
	}

	dec := NewAACDecoder()
	rb := NewAudioRingBuffer(128 * 1024)
	cb := &mockDecoderCallback{}

	err = dec.Decode(context.Background(), bytes.NewReader(data), rb, 512, cb)
	if err != nil {
		t.Fatalf("AACDecoder failed to decode synthetic sine.aac: %v", err)
	}

	if rb.Available() == 0 {
		t.Fatalf("Expected decoded PCM samples in ring buffer, got 0 bytes")
	}

	cb.mu.Lock()
	if cb.sampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", cb.sampleRate)
	}
	if !cb.thresholdReached {
		t.Errorf("Expected thresholdReached to be true")
	}
	cb.mu.Unlock()

	// Verify that audio level calculation produces valid active signal
	pcm := make([]byte, rb.Available())
	_, _ = rb.Read(pcm)
	leftDB, rightDB := dsp.CalculateLevels(pcm)
	if leftDB <= -90 || rightDB <= -90 {
		t.Errorf("Expected active audio levels from sine.aac, got LeftRMS: %f, RightRMS: %f", leftDB, rightDB)
	}
}

func TestVorbisDecoder_EmptyStream(t *testing.T) {
	dec := NewVorbisDecoder()
	rb := NewAudioRingBuffer(1024)

	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(context.Background(), reader, rb, 100, nil)
}

func TestVorbisDecoder_ContextCancel(t *testing.T) {
	dec := NewVorbisDecoder()
	rb := NewAudioRingBuffer(1024)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(ctx, reader, rb, 100, nil)
}

func TestVorbisDecoder_SyntheticFile(t *testing.T) {
	data, err := os.ReadFile("testdata/sine.ogg")
	if err != nil {
		t.Fatalf("Failed to read test vector sine.ogg: %v", err)
	}

	dec := NewVorbisDecoder()
	rb := NewAudioRingBuffer(128 * 1024)
	cb := &mockDecoderCallback{}

	err = dec.Decode(context.Background(), bytes.NewReader(data), rb, 512, cb)
	if err != nil {
		t.Fatalf("VorbisDecoder failed to decode synthetic sine.ogg: %v", err)
	}

	if rb.Available() == 0 {
		t.Fatalf("Expected decoded PCM samples in ring buffer, got 0 bytes")
	}

	cb.mu.Lock()
	if cb.sampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", cb.sampleRate)
	}
	if !cb.thresholdReached {
		t.Errorf("Expected thresholdReached to be true")
	}
	cb.mu.Unlock()

	// Verify that audio level calculation produces valid active signal
	pcm := make([]byte, rb.Available())
	_, _ = rb.Read(pcm)
	leftDB, rightDB := dsp.CalculateLevels(pcm)
	if leftDB <= -90 || rightDB <= -90 {
		t.Errorf("Expected active audio levels from sine.ogg, got LeftRMS: %f, RightRMS: %f", leftDB, rightDB)
	}
}

func TestOpusDecoder_EmptyStream(t *testing.T) {
	dec := NewOpusDecoder()
	rb := NewAudioRingBuffer(1024)

	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(context.Background(), reader, rb, 100, nil)
}

func TestOpusDecoder_ContextCancel(t *testing.T) {
	dec := NewOpusDecoder()
	rb := NewAudioRingBuffer(1024)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader := bytes.NewReader([]byte{})
	_ = dec.Decode(ctx, reader, rb, 100, nil)
}

func TestOpusDecoder_SyntheticFile(t *testing.T) {
	data, err := os.ReadFile("testdata/sine.opus")
	if err != nil {
		t.Fatalf("Failed to read test vector sine.opus: %v", err)
	}

	dec := NewOpusDecoder()
	rb := NewAudioRingBuffer(128 * 1024)
	cb := &mockDecoderCallback{}

	err = dec.Decode(context.Background(), bytes.NewReader(data), rb, 512, cb)
	if err != nil {
		t.Fatalf("OpusDecoder failed to decode synthetic sine.opus: %v", err)
	}

	if rb.Available() == 0 {
		t.Fatalf("Expected decoded PCM samples in ring buffer, got 0 bytes")
	}

	cb.mu.Lock()
	if cb.sampleRate != 48000 {
		t.Errorf("Expected sample rate 48000, got %d", cb.sampleRate)
	}
	if !cb.thresholdReached {
		t.Errorf("Expected thresholdReached to be true")
	}
	cb.mu.Unlock()

	// Verify that audio level calculation produces valid active signal
	pcm := make([]byte, rb.Available())
	_, _ = rb.Read(pcm)
	leftDB, rightDB := dsp.CalculateLevels(pcm)
	if leftDB <= -90 || rightDB <= -90 {
		t.Errorf("Expected active audio levels from sine.opus, got LeftRMS: %f, RightRMS: %f", leftDB, rightDB)
	}
}
