package slimproto

import (
	"bytes"
	"net"
	"testing"
	"time"

	"lautenbacher.net/goleds/audio"
)

func TestSlimProto_EncodeHelo(t *testing.T) {
	mac := net.HardwareAddr{0x00, 0x04, 0x20, 0xaa, 0xbb, 0xcc}
	cfg := HeloConfig{
		MAC:        mac,
		DeviceID:   12,
		Revision:   1,
		PlayerName: "TestPlayer",
	}

	pkt := EncodeHelo(cfg)
	if len(pkt) < 44 {
		t.Fatalf("Packet length too short: %d", len(pkt))
	}
	if string(pkt[0:4]) != "HELO" {
		t.Errorf("Expected opcode HELO, got %s", string(pkt[0:4]))
	}
	if pkt[8] != 12 {
		t.Errorf("Expected DeviceID 12, got %d", pkt[8])
	}
	if !bytes.Equal(pkt[10:16], mac) {
		t.Errorf("MAC mismatch: expected %v, got %v", mac, pkt[10:16])
	}
	if pkt[42] != 'E' || pkt[43] != 'N' {
		t.Errorf("Language mismatch: expected EN, got %c%c", pkt[42], pkt[43])
	}
}

func TestSlimProto_EncodeStat(t *testing.T) {
	pkt := EncodeStat([4]byte{'S', 'T', 'M', 't'}, 65536, 1024, 65536, 2048, 1234, 10, 5678, 0)
	if string(pkt[0:4]) != "STAT" {
		t.Errorf("Expected opcode STAT, got %s", string(pkt[0:4]))
	}
	if string(pkt[8:12]) != "STMt" {
		t.Errorf("Expected event STMt, got %s", string(pkt[8:12]))
	}
}

func TestSlimProto_ParseStrm(t *testing.T) {
	payload := make([]byte, 24)
	payload[0] = 's'
	payload[1] = '1'
	payload[2] = 'p'
	payload[3] = '1'
	payload[4] = '3'
	payload[5] = '2'

	cmd, err := ParseStrm(payload)
	if err != nil {
		t.Fatalf("ParseStrm failed: %v", err)
	}
	if cmd.SubCommand != 's' {
		t.Errorf("Expected SubCommand 's', got %c", cmd.SubCommand)
	}
	if cmd.Format != 'p' {
		t.Errorf("Expected Format 'p', got %c", cmd.Format)
	}
}

func TestSlimProto_ClientHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan bool)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		if n > 0 && string(buf[0:4]) == "HELO" {
			accepted <- true
		}
	}()

	cfg := HeloConfig{
		MAC:      net.HardwareAddr{0x00, 0x04, 0x20, 0xaa, 0xbb, 0xcc},
		DeviceID: 12,
	}

	levels := audio.NewAtomicLevels()
	client := NewClient(ln.Addr().String(), cfg, levels)
	if err := client.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	select {
	case <-accepted:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Handshake timeout")
	}

	_ = client.Stop()
}
