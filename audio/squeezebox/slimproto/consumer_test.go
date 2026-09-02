package slimproto

import (
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/audio"
)

type mockConsumerCallbacks struct {
	mu           sync.Mutex
	state        PlaybackState
	sampleRate   uint32
	startAt      uint32
	pauseFrames  int64
	framesPlayed uint64
	decoderDone  bool
	sentStats    [][4]byte
}

func (m *mockConsumerCallbacks) GetState() PlaybackState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *mockConsumerCallbacks) SetState(s PlaybackState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
}

func (m *mockConsumerCallbacks) GetSampleRate() uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sampleRate == 0 {
		return 44100
	}
	return m.sampleRate
}

func (m *mockConsumerCallbacks) GetStartAt() uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startAt
}

func (m *mockConsumerCallbacks) GetPauseFrames() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pauseFrames
}

func (m *mockConsumerCallbacks) DeductPauseFrames(frames int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if frames >= m.pauseFrames {
		m.pauseFrames = 0
	} else {
		m.pauseFrames -= frames
	}
}

func (m *mockConsumerCallbacks) AddFramesPlayed(frames uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.framesPlayed += frames
}

func (m *mockConsumerCallbacks) IsDecoderDone() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.decoderDone
}

func (m *mockConsumerCallbacks) SendStat(event [4]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentStats = append(m.sentStats, event)
	return nil
}

func TestPacedConsumer_ExactPacing(t *testing.T) {
	rb := NewAudioRingBuffer(65536)
	levels := audio.NewAtomicLevels()
	cb := &mockConsumerCallbacks{
		state:      StateRunning,
		sampleRate: 44100,
	}
	mockClock := NewMockClock(1000, time.Now())

	pc := NewPacedConsumer(PacedConsumerConfig{
		RingBuffer: rb,
		Levels:     levels,
		Clock:      mockClock,
		Callbacks:  cb,
	})

	// Fill ring buffer with 4410 frames (17640 bytes = 100ms of audio)
	audioData := make([]byte, 17640)
	for i := range audioData {
		audioData[i] = 0x20
	}
	_, _ = rb.Write(audioData)

	// Step by 50ms (should consume 2205 frames = 8820 bytes)
	pc.Step(50 * time.Millisecond)

	if cb.framesPlayed != 2205 {
		t.Errorf("Expected 2205 frames played, got %d", cb.framesPlayed)
	}

	left, right, active := levels.Get()
	if !active {
		t.Errorf("Expected active levels, got false")
	}
	if left <= -100 || right <= -100 {
		t.Errorf("Expected valid dB levels, got (%.2f, %.2f)", left, right)
	}

	// Step remaining 50ms
	pc.Step(50 * time.Millisecond)
	if cb.framesPlayed != 4410 {
		t.Errorf("Expected 4410 total frames played, got %d", cb.framesPlayed)
	}
}

func TestPacedConsumer_MicroPause(t *testing.T) {
	rb := NewAudioRingBuffer(65536)
	levels := audio.NewAtomicLevels()
	cb := &mockConsumerCallbacks{
		state:       StateRunning,
		sampleRate:  44100,
		pauseFrames: 441, // 10ms micro pause
	}

	pc := NewPacedConsumer(PacedConsumerConfig{
		RingBuffer: rb,
		Levels:     levels,
		Callbacks:  cb,
	})

	// Step by 5ms
	pc.Step(5 * time.Millisecond)

	if cb.pauseFrames != 221 && cb.pauseFrames != 220 { // ~220.5 frames deducted
		t.Errorf("Expected ~220 pauseFrames remaining, got %d", cb.pauseFrames)
	}
	if cb.framesPlayed != 0 {
		t.Errorf("Expected 0 frames played during pause, got %d", cb.framesPlayed)
	}

	_, _, active := levels.Get()
	if active {
		t.Errorf("Expected active=false during micro pause, got true")
	}
}

func TestPacedConsumer_StartAtSync(t *testing.T) {
	rb := NewAudioRingBuffer(1024)
	levels := audio.NewAtomicLevels()
	mockClock := NewMockClock(500, time.Now())
	cb := &mockConsumerCallbacks{
		state:   StateStartAt,
		startAt: 1000,
	}

	pc := NewPacedConsumer(PacedConsumerConfig{
		RingBuffer: rb,
		Levels:     levels,
		Clock:      mockClock,
		Callbacks:  cb,
	})

	// Step while clock is 500ms (not yet reached startAt 1000ms)
	pc.Step(10 * time.Millisecond)
	if cb.GetState() != StateStartAt {
		t.Errorf("Expected state to remain StateStartAt, got %v", cb.GetState())
	}

	// Advance clock past target
	mockClock.Advance(600 * time.Millisecond) // now 1100ms
	pc.Step(10 * time.Millisecond)

	if cb.GetState() != StateRunning {
		t.Errorf("Expected transition to StateRunning, got %v", cb.GetState())
	}
	if len(cb.sentStats) == 0 || cb.sentStats[0] != [4]byte{'S', 'T', 'M', 's'} {
		t.Errorf("Expected STMs stat event sent on start")
	}
}

func TestPacedConsumer_UnderrunAtEOF(t *testing.T) {
	rb := NewAudioRingBuffer(65536)
	levels := audio.NewAtomicLevels()
	cb := &mockConsumerCallbacks{
		state:       StateRunning,
		sampleRate:  44100,
		decoderDone: true, // EOF reached on input stream
	}

	pc := NewPacedConsumer(PacedConsumerConfig{
		RingBuffer: rb,
		Levels:     levels,
		Callbacks:  cb,
	})

	// Step on empty buffer
	pc.Step(10 * time.Millisecond)

	if cb.GetState() != StateStopped {
		t.Errorf("Expected transition to StateStopped on EOF underrun, got %v", cb.GetState())
	}
	if len(cb.sentStats) == 0 || cb.sentStats[0] != [4]byte{'S', 'T', 'M', 'u'} {
		t.Errorf("Expected STMu stat event sent on EOF underrun")
	}
}
