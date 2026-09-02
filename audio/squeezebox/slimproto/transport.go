package slimproto

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

// CommandHandler receives parsed SlimProto opcodes and payloads from LMS.
type CommandHandler interface {
	HandleCommand(cmd string, payload []byte)
}

// PacketSender transmits binary status and response packets to LMS.
type PacketSender interface {
	SendStat(event [4]byte) error
	SendStatWithTimestamp(event [4]byte, serverTimestamp uint32) error
	SendResp(headers string) error
}

// TransportConfig specifies connection parameters for the SlimProto transport.
type TransportConfig struct {
	ServerAddr string
	HeloConfig HeloConfig
	Handler    CommandHandler
}

// TCPTransport manages the bi-directional TCP control stream to LMS, handling
// handshakes, framed packet dispatching, and automatic reconnection.
type TCPTransport struct {
	serverAddr string
	heloConfig HeloConfig
	handler    CommandHandler

	conn net.Conn
	mu   sync.Mutex
	wg   sync.WaitGroup

	ctx       context.Context
	ctxCancel context.CancelFunc
	running   bool
}

// NewTCPTransport creates an initialized TCPTransport.
func NewTCPTransport(cfg TransportConfig) *TCPTransport {
	return &TCPTransport{
		serverAddr: cfg.ServerAddr,
		heloConfig: cfg.HeloConfig,
		handler:    cfg.Handler,
	}
}

// Start dials LMS, completes the HELO/SETD handshake, and starts the receive loop.
func (t *TCPTransport) Start() error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return nil
	}

	conn, err := net.DialTimeout("tcp", t.serverAddr, 3*time.Second)
	if err != nil {
		t.mu.Unlock()
		return fmt.Errorf("slimproto dial %s: %w", t.serverAddr, err)
	}

	// Send HELO handshake
	heloData := EncodeHelo(t.heloConfig)
	if _, err := conn.Write(heloData); err != nil {
		_ = conn.Close()
		t.mu.Unlock()
		return fmt.Errorf("slimproto send helo: %w", err)
	}

	// Send SETD name frame if configured
	if t.heloConfig.PlayerName != "" {
		setdData := EncodeSetdName(t.heloConfig.PlayerName)
		_, _ = conn.Write(setdData)
	}

	slog.Info("SlimProto connected to LMS and sent HELO/SETD",
		"server", t.serverAddr,
		"mac", t.heloConfig.MAC.String(),
		"name", t.heloConfig.PlayerName)

	t.ctx, t.ctxCancel = context.WithCancel(context.Background())
	t.conn = conn
	t.running = true
	t.mu.Unlock()

	t.wg.Add(1)
	go t.readLoop(conn)

	return nil
}

// Stop closes the connection and terminates all background workers.
func (t *TCPTransport) Stop() error {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = false
	if t.ctxCancel != nil {
		t.ctxCancel()
	}
	if t.conn != nil {
		_ = t.conn.SetDeadline(time.Now())
		_ = t.conn.Close()
	}
	t.mu.Unlock()

	t.wg.Wait()

	t.mu.Lock()
	t.conn = nil
	t.mu.Unlock()
	return nil
}

// Write transmits a raw binary packet to LMS with a 2-second write deadline.
func (t *TCPTransport) Write(data []byte) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()

	if conn == nil {
		return errors.New("slimproto transport not connected")
	}

	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err := conn.Write(data)
	return err
}

func (t *TCPTransport) isRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

func (t *TCPTransport) readLoop(initialConn net.Conn) {
	defer t.wg.Done()

	conn := initialConn
	lenBuf := make([]byte, 2)

	for t.isRunning() {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			if !t.isRunning() {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			slog.Warn("SlimProto read error, attempting reconnect", "server", t.serverAddr, "error", err)

			_ = conn.Close()
			t.mu.Lock()
			if t.conn == conn {
				t.conn = nil
			}
			t.mu.Unlock()

			// Reconnect loop with backoff
			for t.isRunning() {
				select {
				case <-t.ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
				if !t.isRunning() {
					return
				}
				newConn, err := net.DialTimeout("tcp", t.serverAddr, 3*time.Second)
				if err != nil {
					slog.Debug("SlimProto reconnect failed, retrying...", "error", err)
					continue
				}

				// Send HELO handshake
				heloData := EncodeHelo(t.heloConfig)
				if _, err := newConn.Write(heloData); err != nil {
					_ = newConn.Close()
					continue
				}

				// Send SETD name frame if configured
				if t.heloConfig.PlayerName != "" {
					setdData := EncodeSetdName(t.heloConfig.PlayerName)
					_, _ = newConn.Write(setdData)
				}

				t.mu.Lock()
				t.conn = newConn
				t.mu.Unlock()

				conn = newConn
				slog.Info("SlimProto reconnected to LMS successfully", "server", t.serverAddr)
				break
			}
			continue
		}

		totalLen := binary.BigEndian.Uint16(lenBuf)
		if totalLen < 4 {
			slog.Warn("SlimProto received message too short", "length", totalLen)
			continue
		}

		frameBuf := make([]byte, totalLen)
		if _, err := io.ReadFull(conn, frameBuf); err != nil {
			if !t.isRunning() {
				return
			}
			slog.Debug("SlimProto read frame error", "error", err)
			continue
		}

		cmd := string(frameBuf[0:4])
		payload := frameBuf[4:]

		if t.handler != nil {
			t.handler.HandleCommand(cmd, payload)
		}
	}
}
