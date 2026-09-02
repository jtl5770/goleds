package slimproto

import (
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"
)

type mockCommandHandler struct {
	mu       sync.Mutex
	commands []string
	payloads [][]byte
}

func (m *mockCommandHandler) HandleCommand(cmd string, payload []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commands = append(m.commands, cmd)
	m.payloads = append(m.payloads, append([]byte(nil), payload...))
}

func TestTCPTransport_ConnectAndHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer ln.Close()

	handler := &mockCommandHandler{}
	cfg := TransportConfig{
		ServerAddr: ln.Addr().String(),
		HeloConfig: HeloConfig{
			PlayerName: "TestPlayer",
		},
		Handler: handler,
	}

	transport := NewTCPTransport(cfg)

	// Server goroutine
	var serverWg sync.WaitGroup
	serverWg.Add(1)
	go func() {
		defer serverWg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read HELO packet (8 byte framing + payload)
		heloBuf := make([]byte, 256)
		_, _ = conn.Read(heloBuf)

		// Send a framed command from LMS to client: 2-byte length + "audg" opcode + 2 bytes volume
		cmdPayload := []byte("audg\x00\x00")
		lenPrefix := make([]byte, 2)
		binary.BigEndian.PutUint16(lenPrefix, uint16(len(cmdPayload)))

		_, _ = conn.Write(lenPrefix)
		_, _ = conn.Write(cmdPayload)

		// Wait briefly before closing
		time.Sleep(50 * time.Millisecond)
	}()

	if err := transport.Start(); err != nil {
		t.Fatalf("Transport Start failed: %v", err)
	}

	// Allow message receive
	time.Sleep(100 * time.Millisecond)

	handler.mu.Lock()
	if len(handler.commands) == 0 || handler.commands[0] != "audg" {
		t.Errorf("Expected audg command dispatched, got %v", handler.commands)
	}
	handler.mu.Unlock()

	_ = transport.Stop()
	serverWg.Wait()
}
