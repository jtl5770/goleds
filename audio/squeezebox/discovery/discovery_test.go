package discovery

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

func TestEncodeRequest(t *testing.T) {
	req := EncodeRequest("NAME", "IPAD", "JSON")
	expected := []byte("eNAME\x00IPAD\x00JSON\x00")
	if !bytes.Equal(req, expected) {
		t.Fatalf("Expected %q, got %q", expected, req)
	}
}

func TestParseResponse_WithIPAD(t *testing.T) {
	// Construct simulated response: E NAME(4)="Test" IPAD(9)="127.0.0.1" JSON(4)="9002" VERS(5)="8.5.0"
	var buf bytes.Buffer
	buf.WriteByte('E')

	buf.WriteString("NAME")
	buf.WriteByte(byte(len("Test Server")))
	buf.WriteString("Test Server")

	buf.WriteString("IPAD")
	buf.WriteByte(byte(len("192.168.1.50")))
	buf.WriteString("192.168.1.50")

	buf.WriteString("JSON")
	buf.WriteByte(byte(len("9002")))
	buf.WriteString("9002")

	buf.WriteString("VERS")
	buf.WriteByte(byte(len("8.5.1")))
	buf.WriteString("8.5.1")

	info, err := ParseResponse(buf.Bytes(), net.ParseIP("10.0.0.1"))
	if err != nil {
		t.Fatalf("Unexpected parse error: %v", err)
	}

	if info.Host != "192.168.1.50" {
		t.Errorf("Expected Host '192.168.1.50', got %q", info.Host)
	}
	if info.JSONRPCPort != 9002 {
		t.Errorf("Expected JSONRPCPort 9002, got %d", info.JSONRPCPort)
	}
	if info.SlimProtoPort != 3483 {
		t.Errorf("Expected SlimProtoPort 3483, got %d", info.SlimProtoPort)
	}
	if info.Name != "Test Server" {
		t.Errorf("Expected Name 'Test Server', got %q", info.Name)
	}
	if info.Version != "8.5.1" {
		t.Errorf("Expected Version '8.5.1', got %q", info.Version)
	}
}

func TestParseResponse_FallbackToRemoteIP(t *testing.T) {
	// Construct simulated response without IPAD tag
	var buf bytes.Buffer
	buf.WriteByte('E')

	buf.WriteString("NAME")
	buf.WriteByte(byte(len("My LMS")))
	buf.WriteString("My LMS")

	remoteIP := net.ParseIP("192.168.1.99")
	info, err := ParseResponse(buf.Bytes(), remoteIP)
	if err != nil {
		t.Fatalf("Unexpected parse error: %v", err)
	}

	if info.Host != "192.168.1.99" {
		t.Errorf("Expected Host '192.168.1.99' from remoteIP, got %q", info.Host)
	}
	if info.JSONRPCPort != 9000 {
		t.Errorf("Expected default JSONRPCPort 9000, got %d", info.JSONRPCPort)
	}
	if info.SlimProtoPort != 3483 {
		t.Errorf("Expected default SlimProtoPort 3483, got %d", info.SlimProtoPort)
	}
}

func TestDiscoverServer_Mock(t *testing.T) {
	// Start mock UDP responder on loopback
	serverConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to start mock UDP server: %v", err)
	}
	defer serverConn.Close()

	serverPort := serverConn.LocalAddr().(*net.UDPAddr).Port

	stopServer := make(chan struct{})
	defer close(stopServer)

	go func() {
		buf := make([]byte, 512)
		for {
			_ = serverConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, clientAddr, err := serverConn.ReadFromUDP(buf)
			select {
			case <-stopServer:
				return
			default:
			}
			if err != nil || n == 0 {
				continue
			}
			if buf[0] == 'e' {
				var resp bytes.Buffer
				resp.WriteByte('E')
				resp.WriteString("NAME")
				resp.WriteByte(byte(len("Mock LMS")))
				resp.WriteString("Mock LMS")
				resp.WriteString("IPAD")
				resp.WriteByte(byte(len("127.0.0.1")))
				resp.WriteString("127.0.0.1")
				resp.WriteString("JSON")
				resp.WriteByte(byte(len("9000")))
				resp.WriteString("9000")

				_, _ = serverConn.WriteToUDP(resp.Bytes(), clientAddr)
				return
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	info, err := DiscoverServerOnPort(ctx, 2*time.Second, serverPort)
	if err != nil {
		t.Fatalf("Failed to discover mock server: %v", err)
	}

	if info.Host != "127.0.0.1" {
		t.Errorf("Expected Host '127.0.0.1', got %q", info.Host)
	}
	if info.Name != "Mock LMS" {
		t.Errorf("Expected Name 'Mock LMS', got %q", info.Name)
	}
}

func TestDiscoverServer_Timeout(t *testing.T) {
	// Pick an unused local port where no LMS is listening
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := DiscoverServerOnPort(ctx, 50*time.Millisecond, 59999)
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}
}
