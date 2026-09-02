package slimproto

import (
	"context"
	"log/slog"
	"time"

	"lautenbacher.net/goleds/audio"
	"lautenbacher.net/goleds/audio/dsp"
)

// ConsumerCallbacks provides access to player state and telemetry for the PacedConsumer.
type ConsumerCallbacks interface {
	GetState() PlaybackState
	SetState(s PlaybackState)
	GetSampleRate() uint32
	GetStartAt() uint32
	GetPauseFrames() int64
	DeductPauseFrames(frames int64)
	AddFramesPlayed(frames uint64)
	IsDecoderDone() bool
	SendStat(event [4]byte) error
}

// PacedConsumerConfig defines configuration for PacedConsumer.
type PacedConsumerConfig struct {
	TickInterval time.Duration
	RingBuffer   *AudioRingBuffer
	Levels       *audio.AtomicLevels
	Clock        Clock
	Callbacks    ConsumerCallbacks
}

// PacedConsumer drains PCM audio samples from the AudioRingBuffer in real-time,
// synchronizes jiffies timestamps, deducts micro-pause frames, detects underruns,
// and feeds RMS dB audio level measurements into AtomicLevels.
type PacedConsumer struct {
	tickInterval time.Duration
	ringBuffer   *AudioRingBuffer
	levels       *audio.AtomicLevels
	clock        Clock
	callbacks    ConsumerCallbacks

	chunkBuf         []byte
	frameAccumulator float64
}

// NewPacedConsumer creates an initialized PacedConsumer.
func NewPacedConsumer(cfg PacedConsumerConfig) *PacedConsumer {
	interval := cfg.TickInterval
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}
	clock := cfg.Clock
	if clock == nil {
		clock = NewSystemClock()
	}
	return &PacedConsumer{
		tickInterval: interval,
		ringBuffer:   cfg.RingBuffer,
		levels:       cfg.Levels,
		clock:        clock,
		callbacks:    cfg.Callbacks,
		chunkBuf:     make([]byte, 65536),
	}
}

// Run executes the continuous audio consumption loop until ctx is canceled.
func (p *PacedConsumer) Run(ctx context.Context) {
	ticker := time.NewTicker(p.tickInterval)
	defer ticker.Stop()

	lastTime := p.clock.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			dt := now.Sub(lastTime)
			lastTime = now
			p.Step(dt)
		}
	}
}

// Step processes a single time delta of audio consumption.
// Can be called directly by deterministic unit tests with a MockClock.
func (p *PacedConsumer) Step(dt time.Duration) {
	if p.callbacks == nil || p.ringBuffer == nil || p.levels == nil {
		return
	}

	state := p.callbacks.GetState()
	sr := p.callbacks.GetSampleRate()
	if sr == 0 {
		sr = 44100
	}

	switch state {
	case StateStartAt:
		nowMs := p.clock.NowMonotonicMs()
		startAt := p.callbacks.GetStartAt()
		if nowMs >= startAt || (startAt > nowMs && (startAt-nowMs) > 10000) {
			p.callbacks.SetState(StateRunning)
			_ = p.callbacks.SendStat([4]byte{'S', 'T', 'M', 's'})
		}
		p.levels.Set(-100, -100, false)
		p.frameAccumulator = 0

	case StateRunning:
		pauseFrames := p.callbacks.GetPauseFrames()
		if pauseFrames > 0 {
			p.levels.Set(-100, -100, false)
			framesDeducted := int64(float64(sr) * dt.Seconds())
			p.callbacks.DeductPauseFrames(framesDeducted)
			return
		}

		p.frameAccumulator += float64(sr) * dt.Seconds()
		framesToConsume := int(p.frameAccumulator)
		if framesToConsume <= 0 {
			return
		}
		p.frameAccumulator -= float64(framesToConsume)

		bytesToConsume := framesToConsume * 4 // 16-bit stereo = 4 bytes per frame

		if len(p.chunkBuf) < bytesToConsume {
			p.chunkBuf = make([]byte, bytesToConsume)
		}

		n, _ := p.ringBuffer.Read(p.chunkBuf[:bytesToConsume])
		if n > 0 {
			p.callbacks.AddFramesPlayed(uint64(n / 4))
			leftDB, rightDB := dsp.CalculateLevels(p.chunkBuf[:n])
			p.levels.Set(leftDB, rightDB, true)
		} else {
			// Buffer underrun
			if p.callbacks.IsDecoderDone() {
				slog.Info("SlimProto stream playback finished (underrun at EOF)")
				p.callbacks.SetState(StateStopped)
				_ = p.callbacks.SendStat([4]byte{'S', 'T', 'M', 'u'}) // Output underrun (STMu)
			}
			p.levels.Set(-100, -100, false)
		}

	case StateStopped, StateBuffering, StateWaitingStart, StatePaused:
		p.levels.Set(-100, -100, false)
		p.frameAccumulator = 0
	}
}
