package squeezebox

import (
	"testing"
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
