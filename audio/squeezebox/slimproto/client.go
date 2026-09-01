package slimproto

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"lautenbacher.net/goleds/audio"
	"lautenbacher.net/goleds/audio/dsp"
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

// Client handles the SlimProto TCP connection, HTTP audio streaming, FLAC/PCM decoding,
// and real-time paced audio level visualization for GoLEDs conforming to Squeezelite.
type Client struct {
	serverAddr string
	serverHost string
	heloConfig HeloConfig
	levels     *audio.AtomicLevels

	conn       net.Conn
	streamConn net.Conn
	mu         sync.Mutex
	wg         sync.WaitGroup

	streamCtx    context.Context
	streamCancel context.CancelFunc
	streamWg     sync.WaitGroup

	ctx       context.Context
	ctxCancel context.CancelFunc
	running   atomic.Bool

	state       atomic.Int32  // PlaybackState
	startAt     atomic.Uint32 // target jiffies timestamp for unpause ('u')
	autoStart   atomic.Uint32 // autostart mode from strm command ('0'..'3')
	thresholdKB atomic.Uint32 // buffer threshold in KB

	ringBuffer *AudioRingBuffer

	bytesReceived atomic.Uint64
	framesPlayed  atomic.Uint64
	sampleRate    atomic.Uint32
	channels      atomic.Uint32

	decoderDone atomic.Bool
}

// NewClient creates a new Squeezelite-compliant SlimProto Client.
func NewClient(serverAddr string, helo HeloConfig, levels *audio.AtomicLevels) *Client {
	host, _, err := net.SplitHostPort(serverAddr)
	if err != nil {
		host = serverAddr
	}
	if levels == nil {
		levels = audio.NewAtomicLevels()
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		serverAddr: serverAddr,
		serverHost: host,
		heloConfig: helo,
		levels:     levels,
		ringBuffer: NewAudioRingBuffer(2 * 1024 * 1024), // 2 MB PCM buffer (~12s @ 44.1kHz stereo)
		ctx:        ctx,
		ctxCancel:  cancel,
	}
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

// setState sets the playback state atomically and logs the transition.
func (c *Client) setState(s PlaybackState) {
	old := PlaybackState(c.state.Swap(int32(s)))
	if old != s {
		slog.Info("SlimProto state transition", "from", old.String(), "to", s.String())
	}
}

// Start connects to the LMS SlimProto server and starts network and audio loops.
func (c *Client) Start() error {
	conn, err := net.DialTimeout("tcp", c.serverAddr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("slimproto dial %s: %w", c.serverAddr, err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	c.running.Store(true)

	// Send HELO handshake
	heloData := EncodeHelo(c.heloConfig)
	if _, err := conn.Write(heloData); err != nil {
		conn.Close()
		return fmt.Errorf("slimproto send helo: %w", err)
	}

	// Send SETD name frame if configured
	if c.heloConfig.PlayerName != "" {
		setdData := EncodeSetdName(c.heloConfig.PlayerName)
		_, _ = conn.Write(setdData)
	}

	slog.Info("SlimProto connected to LMS and sent HELO/SETD",
		"server", c.serverAddr,
		"mac", c.heloConfig.MAC.String(),
		"name", c.heloConfig.PlayerName)

	c.wg.Add(3)
	go c.readLoop(conn)
	go c.heartbeatLoop()
	go c.audioConsumerLoop()

	return nil
}

// Stop closes the connection and terminates all background workers.
func (c *Client) Stop() error {
	if !c.running.Swap(false) {
		return nil
	}

	if c.ctxCancel != nil {
		c.ctxCancel()
	}

	c.mu.Lock()
	if c.streamCancel != nil {
		c.streamCancel()
	}
	if c.conn != nil {
		_ = c.conn.SetDeadline(time.Now())
		_ = c.conn.Close()
	}
	if c.streamConn != nil {
		_ = c.streamConn.SetDeadline(time.Now())
		_ = c.streamConn.Close()
	}
	c.mu.Unlock()

	c.ringBuffer.Close()

	c.streamWg.Wait()
	c.wg.Wait()

	c.mu.Lock()
	c.conn = nil
	c.streamConn = nil
	c.mu.Unlock()

	c.levels.Set(-100, -100, false)
	slog.Info("SlimProto client stopped")
	return nil
}

// SendStat sends a 53-byte STAT status packet to LMS with current timing and buffer fullness metrics.
func (c *Client) SendStat(event [4]byte) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return errors.New("slimproto not connected")
	}

	jiffies := uint32(time.Now().UnixMilli())
	sr := c.GetSampleRate()
	frames := c.framesPlayed.Load()
	msPlayed := uint32(0)
	if sr > 0 {
		msPlayed = uint32((frames * 1000) / uint64(sr))
	}

	bufCap := uint32(c.ringBuffer.Capacity())
	bufAvail := uint32(c.ringBuffer.Available())
	bytesRecv := c.bytesReceived.Load()

	stat := EncodeStat(event, bufCap, bufAvail, bufCap, bufAvail, bytesRecv, jiffies, msPlayed, 0)

	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write(stat)
	if err != nil {
		slog.Debug("SlimProto failed to send STAT", "event", string(event[:]), "error", err)
	} else {
		slog.Debug("SlimProto sent STAT", "event", string(event[:]), "jiffies", jiffies, "msPlayed", msPlayed, "bufAvail", bufAvail)
	}
	return err
}

// SendResp relays raw HTTP response headers received from the audio streaming server back to LMS.
func (c *Client) SendResp(headers string) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errors.New("slimproto not connected")
	}
	resp := EncodeResp(headers)
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write(resp)
	if err != nil {
		slog.Debug("SlimProto failed to send RESP", "error", err)
	} else {
		slog.Debug("SlimProto sent RESP headers", "len", len(headers))
	}
	return err
}

func (c *Client) heartbeatLoop() {
	defer c.wg.Done()
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

func (c *Client) readLoop(conn net.Conn) {
	defer c.wg.Done()

	lenBuf := make([]byte, 2)
	for c.running.Load() {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			if !c.running.Load() {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			slog.Debug("SlimProto read length prefix error", "error", err)
			return
		}

		totalLen := binary.BigEndian.Uint16(lenBuf)
		if totalLen < 4 {
			slog.Warn("SlimProto received message too short", "length", totalLen)
			continue
		}

		frameBuf := make([]byte, totalLen)
		if _, err := io.ReadFull(conn, frameBuf); err != nil {
			slog.Debug("SlimProto read frame error", "error", err)
			return
		}

		cmd := string(frameBuf[0:4])
		payload := frameBuf[4:]

		c.handleCommand(cmd, payload)
	}
}

func (c *Client) handleCommand(cmd string, payload []byte) {
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
	slog.Info("SlimProto received strm command",
		"subCommand", string(strm.SubCommand),
		"format", string(strm.Format),
		"autostart", string(strm.AutoStart),
		"replayGain_or_jiffies", strm.ReplayGain,
		"thresholdKB", strm.Threshold,
		"serverIP", strm.ServerIP.String(),
		"serverPort", strm.ServerPort)

	switch strm.SubCommand {
	case 's': // Start stream
		c.mu.Lock()
		if c.streamCancel != nil {
			c.streamCancel()
		}
		if c.streamConn != nil {
			_ = c.streamConn.Close()
			c.streamConn = nil
		}
		streamCtx, cancel := context.WithCancel(c.ctx)
		c.streamCtx = streamCtx
		c.streamCancel = cancel
		c.mu.Unlock()

		currentState := c.GetState()
		isTransition := (currentState == StateRunning || currentState == StateStartAt)

		if !isTransition {
			c.ringBuffer.Flush()
			c.framesPlayed.Store(0)
			c.setState(StateBuffering)
		}
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

	case 'p': // Pause stream
		slog.Info("SlimProto strm: PAUSE stream")
		c.setState(StatePaused)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'p'})

	case 'u': // Unpause stream (synchronized group play command)
		startAt := strm.ReplayGain
		slog.Info("SlimProto strm: UNPAUSE stream (Sync Play)", "startAt_jiffies", startAt)
		c.startAt.Store(startAt)

		// Send resume ack (STMr)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'r'})

		currentState := c.GetState()
		if currentState == StateRunning {
			// Already running seamlessly; do not disrupt pacing
			return
		}

		now := uint32(time.Now().UnixMilli())
		if startAt == 0 || now >= startAt || startAt > (now+10000) {
			c.setState(StateRunning)
			_ = c.SendStat([4]byte{'S', 'T', 'M', 's'})
		} else {
			c.setState(StateStartAt)
		}

	case 'q': // Quit / stop stream
		slog.Info("SlimProto strm: QUIT stream")
		c.mu.Lock()
		if c.streamCancel != nil {
			c.streamCancel()
		}
		if c.streamConn != nil {
			_ = c.streamConn.Close()
			c.streamConn = nil
		}
		c.mu.Unlock()

		c.ringBuffer.Flush()
		c.framesPlayed.Store(0)
		c.setState(StateStopped)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'f'})

	case 'f': // Flush buffers
		slog.Debug("SlimProto strm: FLUSH buffers")
		c.mu.Lock()
		if c.streamCancel != nil {
			c.streamCancel()
		}
		if c.streamConn != nil {
			_ = c.streamConn.Close()
			c.streamConn = nil
		}
		c.mu.Unlock()

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

	case 't': // Timestamp tick
		_ = c.SendStat([4]byte{'S', 'T', 'M', 't'})
	}
}

type countingReader struct {
	r       io.Reader
	counter *atomic.Uint64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 {
		cr.counter.Add(uint64(n))
	}
	return n, err
}

func (c *Client) streamDataWorker(ctx context.Context, strm *StrmCommand) {
	targetIP := strm.ServerIP.String()
	if strm.ServerIP == nil || strm.ServerIP.IsUnspecified() || targetIP == "0.0.0.0" {
		targetIP = c.serverHost
	}
	targetPort := strm.ServerPort
	if targetPort == 0 {
		targetPort = 9000
	}

	streamAddr := fmt.Sprintf("%s:%d", targetIP, targetPort)
	slog.Info("SlimProto connecting to audio stream", "addr", streamAddr)

	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", streamAddr)
	if err != nil {
		slog.Error("SlimProto stream dial failed", "addr", streamAddr, "error", err)
		return
	}

	c.mu.Lock()
	c.streamConn = conn
	c.mu.Unlock()

	defer func() {
		_ = conn.Close()
		c.mu.Lock()
		if c.streamConn == conn {
			c.streamConn = nil
		}
		c.mu.Unlock()
	}()

	// Send HTTP GET request
	if _, err := conn.Write([]byte(strm.HTTPHeader)); err != nil {
		slog.Error("SlimProto failed to write HTTP stream header", "error", err)
		return
	}

	bufReader := bufio.NewReader(conn)

	// Read full raw HTTP response headers up to \r\n\r\n conforming to Squeezelite stream.c
	var headerBytes []byte
	for {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			slog.Error("SlimProto failed reading HTTP response headers", "error", err)
			return
		}
		headerBytes = append(headerBytes, []byte(line)...)
		if line == "\r\n" || line == "\n" {
			break
		}
		if len(headerBytes) > 8192 {
			slog.Error("SlimProto HTTP response headers exceeded 8KB")
			return
		}
	}

	// Relay full raw HTTP response headers back to LMS via RESP packet
	_ = c.SendResp(string(headerBytes))
	_ = c.SendStat([4]byte{'S', 'T', 'M', 'h'}) // HTTP headers received (STMh)

	countingStream := &countingReader{r: bufReader, counter: &c.bytesReceived}

	// Calculate threshold in bytes
	threshKB := c.thresholdKB.Load()
	if threshKB == 0 {
		threshKB = 64 // 64 KB default threshold
	}
	thresholdBytes := int(threshKB * 1024)

	switch strm.Format {
	case 'f': // FLAC stream
		c.decodeFLACStream(ctx, bufReader, countingStream, thresholdBytes)
	case 'p': // Raw PCM stream
		c.decodePCMStream(ctx, countingStream, thresholdBytes)
	default:
		slog.Warn("SlimProto received stream format, attempting FLAC decoder", "format", string(strm.Format))
		c.decodeFLACStream(ctx, bufReader, countingStream, thresholdBytes)
	}

	if ctx.Err() == nil {
		c.decoderDone.Store(true)
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'd'}) // Decoder done (STMd)
	}
}

func (c *Client) decodeFLACStream(ctx context.Context, bufReader *bufio.Reader, r io.Reader, thresholdBytes int) {
	peek, _ := bufReader.Peek(4)
	hasFLACContainer := bytes.Equal(peek, []byte("fLaC"))

	sentSTMl := false
	chunkCount := 0

	if hasFLACContainer {
		stream, err := flac.New(r)
		if err != nil {
			slog.Error("SlimProto FLAC container stream init failed", "error", err)
			return
		}
		defer stream.Close()

		if stream.Info != nil {
			sr := stream.Info.SampleRate
			if sr > 0 {
				c.sampleRate.Store(sr)
			}
			c.channels.Store(uint32(stream.Info.NChannels))
			slog.Info("SlimProto container FLAC initialized",
				"sampleRate", sr,
				"channels", stream.Info.NChannels,
				"bitsPerSample", stream.Info.BitsPerSample)
		}

		for c.running.Load() && ctx.Err() == nil {
			f, err := stream.ParseNext()
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Info("SlimProto FLAC stream reached EOF", "framesDecoded", chunkCount)
				} else {
					slog.Debug("SlimProto FLAC parse stopped", "error", err, "framesDecoded", chunkCount)
				}
				break
			}
			if !c.processFLACFrame(f, thresholdBytes, &sentSTMl) {
				break
			}
			chunkCount++
		}
	} else {
		slog.Info("SlimProto decoding raw FLAC frame stream (LMS direct)")
		for c.running.Load() && ctx.Err() == nil {
			f, err := frame.Parse(r)
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Info("SlimProto FLAC stream reached EOF", "framesDecoded", chunkCount)
				} else {
					slog.Debug("SlimProto raw FLAC parse stopped", "error", err, "framesDecoded", chunkCount)
				}
				break
			}
			if !c.processFLACFrame(f, thresholdBytes, &sentSTMl) {
				break
			}
			chunkCount++
		}
	}
}

func (c *Client) processFLACFrame(f *frame.Frame, thresholdBytes int, sentSTMl *bool) bool {
	if len(f.Subframes) == 0 {
		return true
	}
	nSamples := len(f.Subframes[0].Samples)
	if nSamples == 0 {
		return true
	}

	nChannels := len(f.Subframes)
	bitsPerSample := int(f.BitsPerSample)
	sr := uint32(f.SampleRate)
	if sr > 0 {
		c.sampleRate.Store(sr)
	}

	// Convert FLAC subframe samples to 16-bit stereo LittleEndian PCM bytes
	neededBytes := nSamples * 4 // 2 channels * 2 bytes
	pcmBuf := make([]byte, neededBytes)
	idx := 0
	for i := 0; i < nSamples; i++ {
		for ch := 0; ch < 2; ch++ {
			var sample int32
			if ch < nChannels {
				sample = f.Subframes[ch].Samples[i]
			}
			if bitsPerSample > 16 {
				sample >>= (bitsPerSample - 16)
			} else if bitsPerSample < 16 && bitsPerSample > 0 {
				sample <<= (16 - bitsPerSample)
			}
			s16 := int16(sample)
			binary.LittleEndian.PutUint16(pcmBuf[idx:], uint16(s16))
			idx += 2
		}
	}

	// Write to circular PCM ring buffer (blocks if full, providing socket backpressure)
	if _, err := c.ringBuffer.Write(pcmBuf); err != nil {
		slog.Debug("SlimProto ring buffer write interrupted", "error", err)
		return false
	}

	// Trigger buffer loaded (STMl) once threshold is reached
	if !*sentSTMl && c.ringBuffer.Available() >= thresholdBytes {
		*sentSTMl = true
		_ = c.SendStat([4]byte{'S', 'T', 'M', 'l'})
		autoStart := byte(c.autoStart.Load())
		currentState := c.GetState()
		if currentState == StateRunning || currentState == StateStartAt {
			// Continuous / gapless playback: output is already actively consuming
			// remaining frames from previous track and transition buffer.
		} else if autoStart == '1' {
			c.setState(StateRunning)
			_ = c.SendStat([4]byte{'S', 'T', 'M', 's'})
		} else {
			c.setState(StateWaitingStart)
		}
	}
	return true
}

func (c *Client) decodePCMStream(ctx context.Context, r io.Reader, thresholdBytes int) {
	buf := make([]byte, 4096)
	sentSTMl := false

	for c.running.Load() && ctx.Err() == nil {
		n, err := r.Read(buf)
		if n > 0 {
			if _, werr := c.ringBuffer.Write(buf[:n]); werr != nil {
				break
			}
			if !sentSTMl && c.ringBuffer.Available() >= thresholdBytes {
				sentSTMl = true
				_ = c.SendStat([4]byte{'S', 'T', 'M', 'l'})
				autoStart := byte(c.autoStart.Load())
				currentState := c.GetState()
				if currentState == StateRunning || currentState == StateStartAt {
					// Continuous / gapless playback: output is already running
				} else if autoStart == '1' {
					c.setState(StateRunning)
					_ = c.SendStat([4]byte{'S', 'T', 'M', 's'})
				} else {
					c.setState(StateWaitingStart)
				}
			}
		}
		if err != nil {
			break
		}
	}
}

// audioConsumerLoop consumes PCM samples from ringBuffer in real-time pace, updates
// framesPlayed, computes Left/Right dB levels, and writes to AtomicLevels for the VU meter.
func (c *Client) audioConsumerLoop() {
	defer c.wg.Done()

	const tickInterval = 10 * time.Millisecond
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	lastTime := time.Now()
	chunkBuf := make([]byte, 65536)

	for {
		select {
		case <-c.ctx.Done():
			return
		case now := <-ticker.C:
			if !c.running.Load() {
				return
			}

			dt := now.Sub(lastTime)
			lastTime = now

			state := c.GetState()
			sr := c.GetSampleRate()
			if sr == 0 {
				sr = 44100
			}

			switch state {
			case StateStartAt:
				nowMs := uint32(now.UnixMilli())
				startAt := c.startAt.Load()
				if nowMs >= startAt || startAt > (nowMs+10000) {
					c.setState(StateRunning)
					_ = c.SendStat([4]byte{'S', 'T', 'M', 's'})
				}
				c.levels.Set(-100, -100, false)

			case StateRunning:
				framesToConsume := int(float64(sr) * dt.Seconds())
				if framesToConsume <= 0 {
					framesToConsume = int(sr / 100) // ~10ms fallback
				}
				bytesToConsume := framesToConsume * 4 // 16-bit stereo

				if len(chunkBuf) < bytesToConsume {
					chunkBuf = make([]byte, bytesToConsume)
				}

				n, _ := c.ringBuffer.Read(chunkBuf[:bytesToConsume])
				if n > 0 {
					c.framesPlayed.Add(uint64(n / 4))
					leftDB, rightDB := dsp.CalculateLevels(chunkBuf[:n])
					c.levels.Set(leftDB, rightDB, true)
				} else {
					// Buffer underrun
					if c.decoderDone.Load() {
						slog.Info("SlimProto stream playback finished (underrun at EOF)")
						c.setState(StateStopped)
						_ = c.SendStat([4]byte{'S', 'T', 'M', 'u'}) // Output underrun (STMu)
					}
					c.levels.Set(-100, -100, false)
				}

			case StateStopped, StateBuffering, StateWaitingStart, StatePaused:
				c.levels.Set(-100, -100, false)
			}
		}
	}
}
