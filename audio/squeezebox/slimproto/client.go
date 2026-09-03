package slimproto

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"lautenbacher.net/goleds/audio"
)

// PlaybackState represents the player's internal state machine matching Squeezelite.
type PlaybackState int32

const (
	StateStopped PlaybackState = iota
	StateBuffering
	StateWaitingStart
	StateStartAt
	StateRunning
	StatePaused
)

func (s PlaybackState) String() string {
	switch s {
	case StateStopped:
		return "STOPPED"
	case StateBuffering:
		return "BUFFERING"
	case StateWaitingStart:
		return "WAITING_START"
	case StateStartAt:
		return "START_AT"
	case StateRunning:
		return "RUNNING"
	case StatePaused:
		return "PAUSED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// Client coordinates SlimProto transport, audio fetching, codec decoding,
// and real-time paced audio level visualization for GoLEDs conforming to Squeezelite.
type Client struct {
	serverHost string
	heloConfig HeloConfig
	levels     *audio.AtomicLevels

	transport   *TCPTransport
	fetcher     StreamFetcher
	flacDecoder *FLACDecoder
	ringBuffer  *AudioRingBuffer
	clock       Clock
	consumer    *PacedConsumer

	mu           sync.Mutex
	wg           sync.WaitGroup
	streamCancel context.CancelFunc
	streamWg     sync.WaitGroup

	ctx       context.Context
	ctxCancel context.CancelFunc
	running   atomic.Bool

	state         atomic.Int32  // PlaybackState
	startAt       atomic.Uint32 // target jiffies timestamp for unpause ('u')
	autoStart     atomic.Uint32 // autostart mode from strm command ('0'..'3')
	thresholdKB   atomic.Uint32 // buffer threshold in KB
	pauseFrames   atomic.Int64  // sync micro-pause frames requested by LMS
	currentFormat atomic.Uint32 // active stream format byte ('f', 'm', 'p', 'a', 'o', 'u')

	bytesReceived atomic.Uint64
	framesPlayed  atomic.Uint64
	sampleRate    atomic.Uint32
	channels      atomic.Uint32

	decoderDone atomic.Bool
}

// NewClient creates a new Squeezelite-compliant SlimProto Client orchestrator.
func NewClient(serverAddr string, heloConfig HeloConfig, levels *audio.AtomicLevels) *Client {
	host, _, err := net.SplitHostPort(serverAddr)
	if err != nil {
		host = serverAddr
	}
	if levels == nil {
		levels = audio.NewAtomicLevels()
	}

	rb := NewAudioRingBuffer(2 * 1024 * 1024) // 2 MB PCM buffer (~12s @ 44.1kHz stereo)
	clock := NewSystemClock()
	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		serverHost:  host,
		heloConfig:  heloConfig,
		levels:      levels,
		fetcher:     NewHTTPStreamer(5 * time.Second),
		flacDecoder: NewFLACDecoder(),
		ringBuffer:  rb,
		clock:       clock,
		ctx:         ctx,
		ctxCancel:   cancel,
	}

	c.consumer = NewPacedConsumer(PacedConsumerConfig{
		TickInterval: 10 * time.Millisecond,
		RingBuffer:   rb,
		Levels:       levels,
		Clock:        clock,
		Callbacks:    c,
	})

	c.transport = NewTCPTransport(TransportConfig{
		ServerAddr: serverAddr,
		HeloConfig: heloConfig,
		Handler:    c,
	})

	c.sampleRate.Store(44100)
	c.channels.Store(2)
	c.state.Store(int32(StateStopped))
	return c
}

// GetSampleRate returns the current stream sample rate dynamically.
func (c *Client) GetSampleRate() uint32 {
	sr := c.sampleRate.Load()
	if sr == 0 {
		return 44100
	}
	return sr
}

// GetState returns the current playback state.
func (c *Client) GetState() PlaybackState {
	return PlaybackState(c.state.Load())
}

// SetState sets the playback state atomically and logs the transition.
func (c *Client) SetState(s PlaybackState) {
	old := PlaybackState(c.state.Swap(int32(s)))
	if old != s {
		slog.Info("SlimProto state transition", "from", old.String(), "to", s.String())
	}
}

// GetStartAt returns the target jiffies start timestamp for unpause.
func (c *Client) GetStartAt() uint32 {
	return c.startAt.Load()
}

// GetPauseFrames returns the remaining micro-pause frames.
func (c *Client) GetPauseFrames() int64 {
	return c.pauseFrames.Load()
}

// DeductPauseFrames decrements the micro-pause frame counter.
func (c *Client) DeductPauseFrames(frames int64) {
	if frames >= c.pauseFrames.Load() {
		c.pauseFrames.Store(0)
	} else {
		c.pauseFrames.Add(-frames)
	}
}

// AddFramesPlayed increments the cumulative frames played metric.
func (c *Client) AddFramesPlayed(frames uint64) {
	c.framesPlayed.Add(frames)
}

// IsDecoderDone returns true if the input stream reached EOF.
func (c *Client) IsDecoderDone() bool {
	return c.decoderDone.Load()
}

// OnSampleRateChanged is invoked by Decoders when stream parameters are detected.
func (c *Client) OnSampleRateChanged(sr uint32, channels uint32) {
	if sr > 0 {
		c.sampleRate.Store(sr)
	}
	if channels > 0 {
		c.channels.Store(channels)
	}
}

// OnThresholdReached is invoked by Decoders once buffer threshold is loaded.
func (c *Client) OnThresholdReached() {
	_ = c.SendStat([4]byte{'S', 'T', 'M', 'l'})
	autoStart := byte(c.autoStart.Load())
	currentState := c.GetState()
	if currentState == StateRunning || currentState == StateStartAt {
		// Continuous / gapless playback: output is already actively consuming
	} else if autoStart == '1' || autoStart == '3' {
		c.SetState(StateRunning)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 's'})
	} else {
		// autostart == '0' or '2': wait for LMS 'strm u' synchronized unpause
		c.SetState(StateWaitingStart)
	}
}

// Start connects to the LMS SlimProto server and starts network and audio loops.
func (c *Client) Start() error {
	if err := c.transport.Start(); err != nil {
		return err
	}

	c.mu.Lock()
	if c.ctx == nil || c.ctx.Err() != nil {
		c.ctx, c.ctxCancel = context.WithCancel(context.Background())
	}
	c.mu.Unlock()

	c.running.Store(true)

	c.wg.Add(2)
	go func() {
		defer c.wg.Done()
		c.heartbeatLoop()
	}()
	go func() {
		defer c.wg.Done()
		c.consumer.Run(c.ctx)
	}()

	return nil
}

// Stop closes the connection and terminates all background workers.
func (c *Client) Stop() error {
	if !c.running.Swap(false) {
		return nil
	}

	c.mu.Lock()
	if c.ctxCancel != nil {
		c.ctxCancel()
	}
	if c.streamCancel != nil {
		c.streamCancel()
	}
	c.mu.Unlock()

	_ = c.transport.Stop()
	c.ringBuffer.Close()

	c.streamWg.Wait()
	c.wg.Wait()

	c.levels.Set(-100, -100, false)
	slog.Info("SlimProto client stopped")
	return nil
}

// SendStat sends a 53-byte STAT status packet to LMS with current timing and buffer fullness metrics.
func (c *Client) SendStat(event [4]byte) error {
	return c.SendStatWithTimestamp(event, 0)
}

// SendStatWithTimestamp sends a STAT packet echoing LMS's serverTimestamp for round-trip latency tracking.
func (c *Client) SendStatWithTimestamp(event [4]byte, serverTimestamp uint32) error {
	jiffies := c.clock.NowMonotonicMs()
	sr := c.GetSampleRate()
	frames := c.framesPlayed.Load()
	msPlayed := uint32(0)
	if sr > 0 {
		msPlayed = uint32((frames * 1000) / uint64(sr))
	}

	bufCap := uint32(c.ringBuffer.Capacity())
	bufAvail := uint32(c.ringBuffer.Available())
	bytesRecv := c.bytesReceived.Load()

	stat := EncodeStat(event, bufCap, bufAvail, bufCap, bufAvail, bytesRecv, jiffies, msPlayed, serverTimestamp)
	err := c.transport.Write(stat)
	if err != nil {
		slog.Debug("SlimProto failed to send STAT", "event", string(event[:]), "error", err)
	} else {
		slog.Debug("SlimProto sent STAT", "event", string(event[:]), "jiffies", jiffies, "msPlayed", msPlayed, "bufAvail", bufAvail, "serverTimestamp", serverTimestamp)
	}
	return err
}

// SendResp relays raw HTTP response headers received from the audio streaming server back to LMS.
func (c *Client) SendResp(headers string) error {
	resp := EncodeResp(headers)
	err := c.transport.Write(resp)
	if err != nil {
		slog.Debug("SlimProto failed to send RESP", "error", err)
	} else {
		slog.Debug("SlimProto sent RESP headers", "len", len(headers))
	}
	return err
}

func (c *Client) heartbeatLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if !c.running.Load() {
				return
			}
			_ = c.SendStat([4]byte{'S', 'T', 'M', 't'})
		}
	}
}

// HandleCommand implements CommandHandler dispatched by TCPTransport.
func (c *Client) HandleCommand(cmd string, payload []byte) {
	switch cmd {
	case "strm":
		strm, err := ParseStrm(payload)
		if err != nil {
			slog.Warn("Failed to parse strm command", "error", err)
			return
		}
		c.handleStrm(strm)

	case "audg":
		// Audio gain / volume setting
	case "setd":
		// Setup device display
	default:
		slog.Debug("SlimProto received unhandled opcode", "cmd", cmd, "length", len(payload))
	}
}

func (c *Client) handleStrm(strm *StrmCommand) {
	switch strm.SubCommand {
	case 's': // Start stream
		c.currentFormat.Store(uint32(strm.Format))
		slog.Info("SlimProto received strm command: START",
			"subCommand", "s",
			"format", string(strm.Format),
			"autostart", string(strm.AutoStart),
			"thresholdKB", strm.Threshold,
			"serverIP", strm.ServerIP.String(),
			"serverPort", strm.ServerPort)

		c.mu.Lock()
		if c.streamCancel != nil {
			c.streamCancel()
		}
		streamCtx, cancel := context.WithCancel(c.ctx)
		c.streamCancel = cancel
		c.mu.Unlock()

		currentState := c.GetState()
		isTransition := (currentState == StateRunning || currentState == StateStartAt)

		if !isTransition {
			c.ringBuffer.Flush()
			c.framesPlayed.Store(0)
			c.SetState(StateBuffering)
		}
		c.pauseFrames.Store(0)
		c.autoStart.Store(uint32(strm.AutoStart))
		c.thresholdKB.Store(uint32(strm.Threshold))
		c.decoderDone.Store(false)

		// Send flush ack (STMf)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'f'})

		// Normalize HTTP request header
		header := strm.HTTPHeader
		if header == "" {
			header = fmt.Sprintf("GET /stream.mp3?player=%s HTTP/1.0\r\n\r\n", c.heloConfig.MAC.String())
		} else {
			header = strings.ReplaceAll(header, `\r\n`, "\r\n")
			if !strings.HasPrefix(header, "GET ") && !strings.HasPrefix(header, "POST ") && !strings.HasPrefix(header, "HEAD ") {
				header = "GET " + header
			}
			if !strings.HasSuffix(header, "\r\n\r\n") {
				header = strings.TrimRight(header, "\r\n") + "\r\n\r\n"
			}
		}
		strm.HTTPHeader = header

		c.streamWg.Add(1)
		go func() {
			defer c.streamWg.Done()
			c.streamDataWorker(streamCtx, strm)
		}()

	case 'p': // Pause stream or temporary sync pause
		intervalMs := strm.ReplayGain
		if intervalMs > 0 {
			pauseFrames := (uint64(intervalMs) * uint64(c.GetSampleRate())) / 1000
			slog.Info("SlimProto strm: SYNC PAUSE (delay frames)", "intervalMs", intervalMs, "pauseFrames", pauseFrames)
			c.pauseFrames.Store(int64(pauseFrames))
		} else {
			slog.Info("SlimProto strm: PAUSE stream")
			c.SetState(StatePaused)
			_ = c.SendStat([4]byte{'S', 'T', 'M', 'p'})
		}

	case 'u': // Unpause stream (synchronized group play command)
		startAt := strm.ReplayGain
		activeFmt := byte(c.currentFormat.Load())
		if activeFmt == 0 {
			activeFmt = strm.Format
		}
		slog.Info("SlimProto strm: UNPAUSE stream (Sync Play)",
			"startAt_jiffies", startAt,
			"format", string(activeFmt))
		c.startAt.Store(startAt)

		// Send resume ack (STMr)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'r'})

		now := c.clock.NowMonotonicMs()
		if startAt == 0 || now >= startAt || (startAt > now && (startAt-now) > 10000) {
			c.SetState(StateRunning)
			_ = c.SendStat([4]byte{'S', 'T', 'M', 's'})
		} else {
			c.SetState(StateStartAt)
		}

	case 'q': // Quit / stop stream
		c.currentFormat.Store(0)
		slog.Info("SlimProto strm: QUIT stream")
		c.mu.Lock()
		if c.streamCancel != nil {
			c.streamCancel()
		}
		c.mu.Unlock()

		c.pauseFrames.Store(0)
		c.ringBuffer.Flush()
		c.framesPlayed.Store(0)
		c.SetState(StateStopped)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'f'})

	case 'f': // Flush buffers
		slog.Debug("SlimProto strm: FLUSH buffers")
		c.mu.Lock()
		if c.streamCancel != nil {
			c.streamCancel()
		}
		c.mu.Unlock()

		c.pauseFrames.Store(0)
		c.ringBuffer.Flush()
		c.framesPlayed.Store(0)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'f'})

	case 'a': // Skip ahead
		skipFrames := uint64(strm.ReplayGain) * uint64(c.GetSampleRate()) / 1000
		slog.Info("SlimProto strm: SKIP AHEAD", "skipFrames", skipFrames)
		skipBytes := int(skipFrames * 4)
		discardBuf := make([]byte, min(skipBytes, 65536))
		for skipBytes > 0 {
			toRead := min(skipBytes, len(discardBuf))
			n, _ := c.ringBuffer.Read(discardBuf[:toRead])
			if n == 0 {
				break
			}
			skipBytes -= n
		}
		c.framesPlayed.Add(skipFrames)

	case 't': // Timestamp tick with latency tracking
		slog.Debug("SlimProto strm: TICK latency ping", "timestamp", strm.ReplayGain)
		_ = c.SendStatWithTimestamp([4]byte{'S', 'T', 'M', 't'}, strm.ReplayGain)
	}
}

func (c *Client) streamDataWorker(ctx context.Context, strm *StrmCommand) {
	meta, err := c.fetcher.Fetch(ctx, strm.ServerIP, strm.ServerPort, c.serverHost, strm.HTTPHeader, &c.bytesReceived)
	if err != nil {
		slog.Error("SlimProto stream fetch failed", "error", err)
		return
	}
	defer meta.Conn.Close()

	// Signal stream connected (STMc) conforming to Squeezelite slimproto.c
	_ = c.SendStat([4]byte{'S', 'T', 'M', 'c'})

	// Relay full raw HTTP response headers back to LMS via RESP packet
	_ = c.SendResp(meta.Headers)
	_ = c.SendStat([4]byte{'S', 'T', 'M', 'h'}) // HTTP headers received (STMh)

	// Calculate threshold in bytes
	threshKB := c.thresholdKB.Load()
	if threshKB == 0 {
		threshKB = 64 // 64 KB default threshold
	}
	thresholdBytes := int(threshKB * 1024)

	var decoder Decoder
	switch strm.Format {
	case 'f':
		decoder = c.flacDecoder
	case 'p':
		decoder = NewPCMDecoder(ParsePCMConfig(strm.PCMSampleRate, strm.PCMSampleSize, strm.PCMChannels, strm.PCMEndianness))
	case 'm':
		decoder = NewMP3Decoder()
	case 'a':
		decoder = NewAACDecoder()
	case 'o':
		decoder = NewVorbisDecoder()
	case 'u':
		decoder = NewOpusDecoder()
	default:
		slog.Warn("SlimProto received stream format, attempting FLAC decoder", "format", string(strm.Format))
		decoder = c.flacDecoder
	}

	if err := decoder.Decode(ctx, meta.BodyReader, c.ringBuffer, thresholdBytes, c); err != nil && ctx.Err() == nil {
		slog.Debug("SlimProto decoder completed with status", "error", err)
	}

	if ctx.Err() == nil {
		c.decoderDone.Store(true)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'd'}) // Decoder done (STMd)
	}
}
