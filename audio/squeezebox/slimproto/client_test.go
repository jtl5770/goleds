package slimproto

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/audio"
)

func TestSlimProto_EncodeHelo(t *testing.T) {
	mac, _ := net.ParseMAC("00:04:20:11:22:33")
	cfg := HeloConfig{
		DeviceID:     12, // SqueezePlay
		Revision:     1,
		MAC:          mac,
		Capabilities: "Model=squeezeplay,ModelName=SqueezePlay,flac,pcm",
		PlayerName:   "TestPlayer",
	}

	pkt := EncodeHelo(cfg)
	if len(pkt) < 44 {
		t.Fatalf("Expected HELO packet >= 44 bytes, got %d", len(pkt))
	}
	if !bytes.Equal(pkt[0:4], OpHelo[:]) {
		t.Errorf("Expected opcode HELO, got %q", string(pkt[0:4]))
	}
	if pkt[8] != 12 {
		t.Errorf("Expected deviceID 12, got %d", pkt[8])
	}
	if !bytes.Equal(pkt[10:16], mac) {
		t.Errorf("Expected MAC %v, got %v", mac, pkt[10:16])
	}
}

func TestSlimProto_EncodeStat(t *testing.T) {
	event := [4]byte{'S', 'T', 'M', 'l'}
	pkt := EncodeStat(event, 65536, 32768, 65536, 32768, 100000, 123456, 54321, 0)
	// 8 bytes framing (4 bytes opcode + 4 bytes length) + 53 bytes STAT payload = 61 bytes
	if len(pkt) != 61 {
		t.Fatalf("Expected STAT packet 61 bytes (8+53), got %d", len(pkt))
	}
	if !bytes.Equal(pkt[0:4], OpStat[:]) {
		t.Errorf("Expected opcode STAT, got %q", string(pkt[0:4]))
	}
	if !bytes.Equal(pkt[8:12], event[:]) {
		t.Errorf("Expected event STMl, got %q", string(pkt[8:12]))
	}
}

func TestSlimProto_ParseStrm(t *testing.T) {
	// Build mock 24-byte strm payload
	payload := make([]byte, 24)
	payload[0] = 's'                                     // start
	payload[1] = '2'                                     // autostart
	payload[2] = 'f'                                     // flac
	payload[3] = 16                                      // pcm sample size
	payload[4] = 4                                       // pcm sample rate
	payload[5] = 2                                       // pcm channels
	payload[6] = 0                                       // endian
	payload[7] = 64                                      // threshold KB
	payload[19] = 100                                    // server port (binary 9000 -> 0x2328)
	payload[18] = 0x23
	payload[19] = 0x28
	copy(payload[20:24], net.ParseIP("192.168.1.50").To4())

	strm, err := ParseStrm(payload)
	if err != nil {
		t.Fatalf("Failed to parse valid strm payload: %v", err)
	}

	if strm.SubCommand != 's' {
		t.Errorf("Expected subCommand 's', got '%c'", strm.SubCommand)
	}
	if strm.Format != 'f' {
		t.Errorf("Expected format 'f', got '%c'", strm.Format)
	}
	if strm.AutoStart != '2' {
		t.Errorf("Expected autostart '2', got '%c'", strm.AutoStart)
	}
	if strm.Threshold != 64 {
		t.Errorf("Expected threshold 64, got %d", strm.Threshold)
	}
	if strm.ServerPort != 9000 {
		t.Errorf("Expected serverPort 9000, got %d", strm.ServerPort)
	}
	if strm.ServerIP.String() != "192.168.1.50" {
		t.Errorf("Expected serverIP 192.168.1.50, got %s", strm.ServerIP.String())
	}
}

func TestSlimProto_ClientHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock LMS: %v", err)
	}
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read HELO packet
		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil || n < 8 {
			t.Errorf("Mock LMS read HELO failed: %v", err)
		}
	}()

	mac, _ := net.ParseMAC("00:04:20:aa:bb:cc")
	cfg := HeloConfig{
		MAC:      mac,
		DeviceID: 12,
	}
	levels := audio.NewAtomicLevels()
	client := NewClient(ln.Addr().String(), cfg, levels)

	if err := client.Start(); err != nil {
		t.Fatalf("Client Start failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	_ = client.Stop()
	wg.Wait()
}

func TestSlimProto_TrackTransitionPreservesBuffer(t *testing.T) {
	mac, _ := net.ParseMAC("00:04:20:aa:bb:cc")
	cfg := HeloConfig{
		MAC:      mac,
		DeviceID: 12,
	}
	levels := audio.NewAtomicLevels()
	client := NewClient("127.0.0.1:3483", cfg, levels)

	// Pre-fill buffer with sample data and set state to Running
	sampleData := make([]byte, 1024)
	for i := range sampleData {
		sampleData[i] = byte(i % 256)
	}
	_, _ = client.ringBuffer.Write(sampleData)
	client.SetState(StateRunning)

	// Simulate incoming 'strm s' for next track transition (e.g. gapless)
	strm := &StrmCommand{
		SubCommand: 's',
		AutoStart:  '2',
		Format:     'f',
		ServerIP:   net.ParseIP("127.0.0.1"),
		ServerPort: 9000,
	}
	client.handleStrm(strm)

	// State should remain StateRunning
	if client.GetState() != StateRunning {
		t.Errorf("Expected state to remain StateRunning, got %v", client.GetState())
	}

	// Ring buffer should still retain the 1024 bytes
	if client.ringBuffer.Available() != 1024 {
		t.Errorf("Expected ring buffer to retain 1024 bytes across track transition, got %d", client.ringBuffer.Available())
	}
}
