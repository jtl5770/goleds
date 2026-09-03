package squeezebox

import (
	"net"
	"testing"
	"time"
)

func TestGeneratePlayerMAC(t *testing.T) {
	mac := GeneratePlayerMAC()
	if len(mac) != 6 {
		t.Fatalf("Expected 6 bytes MAC, got %d", len(mac))
	}
	if mac[0] != 0x00 || mac[1] != 0x04 || mac[2] != 0x20 || mac[3] != 0xee {
		t.Errorf("Expected prefix 00:04:20:ee, got %s", mac.String())
	}
}

func TestSqueezeboxAudioProvider_Init(t *testing.T) {
	cfg := Config{
		Server:        "127.0.0.1",
		SlimProtoPort: 3483,
		JSONRPCPort:   9000,
		PlayerMAC:     "auto",
		PlayerName:    "Test VU",
	}

	provider, err := NewSqueezeboxAudioProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	leftDB, rightDB, playing := provider.GetLevels()
	if playing {
		t.Errorf("Expected initial playing to be false")
	}
	if leftDB != -100 || rightDB != -100 {
		t.Errorf("Expected initial levels to be -100, got %f, %f", leftDB, rightDB)
	}
}

func TestSqueezeboxAudioProvider_StartStopLifecycle(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer ln.Close()

	tcpAddr := ln.Addr().(*net.TCPAddr)

	cfg := Config{
		Server:        "127.0.0.1",
		SlimProtoPort: tcpAddr.Port,
		JSONRPCPort:   9000,
		PlayerMAC:     "00:04:20:ee:11:22",
		PlayerName:    "Test Lifecycle",
		AutoSync:      false,
		PollInterval:  50 * time.Millisecond,
	}

	provider, err := NewSqueezeboxAudioProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	go func() {
		conn, err := ln.Accept()
		if err == nil {
			defer conn.Close()
			buf := make([]byte, 256)
			_, _ = conn.Read(buf)
		}
	}()

	if err := provider.Start(); err != nil {
		t.Fatalf("Failed to start provider: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	if err := provider.Stop(); err != nil {
		t.Fatalf("Failed to stop provider: %v", err)
	}
}
