package slimproto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// Standard SlimProto opcodes
var (
	OpHelo = [4]byte{'H', 'E', 'L', 'O'}
	OpStat = [4]byte{'S', 'T', 'A', 'T'}
	OpResp = [4]byte{'R', 'E', 'S', 'P'}
	OpSetd = [4]byte{'S', 'E', 'T', 'D'}
	OpStrm = [4]byte{'s', 't', 'r', 'm'}
)

// HeloConfig holds configuration for the initial HELO handshake packet.
type HeloConfig struct {
	DeviceID     uint8
	Revision     uint8
	MAC          net.HardwareAddr
	Capabilities string
	PlayerName   string
}

// EncodeHelo creates a standard HELO packet for handshake with LMS conforming to Squeezelite.
// Client -> Server framing: 4 bytes opcode ("HELO"), 4 bytes uint32 length, N bytes payload.
// Payload layout (36 bytes fixed header + variable-length capabilities):
// [0]     device_id (1 byte, 12 = SqueezePlay / SqueezeSlave)
// [1]     revision (1 byte)
// [2..8]  mac (6 bytes)
// [8..24] uuid (16 bytes, zeroed)
// [24..26] wlan_channel_list (2 bytes, 0)
// [26..30] bytes_received_H (4 bytes, 0)
// [30..34] bytes_received_L (4 bytes, 0)
// [34..36] language (2 bytes, "EN")
// [36..]  capabilities string
func EncodeHelo(cfg HeloConfig) []byte {
	caps := cfg.Capabilities
	if caps == "" {
		modelName := cfg.PlayerName
		if modelName == "" {
			modelName = "GoLEDs VU"
		}
		caps = fmt.Sprintf("Model=squeezelite,ModelName=%s,MaxSampleRate=384000,AccuratePlayPoints=1,HasDigitalOut=1,HasPolarityInversion=1,Firmware=v1.9.9-1414,flc,pcm,mp3,aac,ogg,ops", modelName)
	}
	payloadLen := 36 + len(caps)
	buf := make([]byte, 8+payloadLen)

	copy(buf[0:4], OpHelo[:])
	binary.BigEndian.PutUint32(buf[4:8], uint32(payloadLen))

	offset := 8
	buf[offset] = cfg.DeviceID   // [0] device_id (12 = SqueezePlay)
	buf[offset+1] = cfg.Revision // [1] revision
	offset += 2

	// [2..8] 6 bytes MAC address
	mac := cfg.MAC
	if len(mac) < 6 {
		mac = net.HardwareAddr{0x00, 0x04, 0x20, 0x00, 0x00, 0x01}
	}
	copy(buf[offset:offset+6], mac[:6])
	offset += 6

	// [8..24] 16 bytes UUID (zeroed)
	offset += 16

	// [24..26] wlan channel list (2 bytes)
	binary.BigEndian.PutUint16(buf[offset:offset+2], 0)
	offset += 2

	// [26..30] bytes_received_H (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:offset+4], 0)
	offset += 4

	// [30..34] bytes_received_L (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:offset+4], 0)
	offset += 4

	// [34..36] 2 bytes language code ("EN")
	copy(buf[offset:offset+2], []byte("EN"))
	offset += 2

	// [36..] variable-length capability strings
	copy(buf[offset:], []byte(caps))

	return buf
}

// EncodeSetdName creates a SETD packet for setting player display/name (id=0).
// Squeezelite sends id=0 followed by null-terminated player name string.
func EncodeSetdName(name string) []byte {
	payloadLen := 1 + len(name) + 1
	buf := make([]byte, 8+payloadLen)
	copy(buf[0:4], OpSetd[:])
	binary.BigEndian.PutUint32(buf[4:8], uint32(payloadLen))
	buf[8] = 0x00
	copy(buf[9:9+len(name)], name)
	buf[8+payloadLen-1] = 0x00
	return buf
}

// EncodeResp creates a standard RESP packet relaying HTTP response headers to LMS.
func EncodeResp(header string) []byte {
	payloadLen := len(header)
	buf := make([]byte, 8+payloadLen)
	copy(buf[0:4], OpResp[:])
	binary.BigEndian.PutUint32(buf[4:8], uint32(payloadLen))
	copy(buf[8:], header)
	return buf
}

// EncodeStat creates a 53-byte standard Squeezelite/SlimProto STAT packet.
// Client -> Server framing: 4 bytes opcode ("STAT"), 4 bytes uint32 length (53), 53 bytes payload.
// [0..4]   event (4 bytes ASCII, e.g. "STMt", "STMh", "STMl", "STMs", "STMp", "STMr", "STMu", "STMf")
// [4]      num_crlf (1 byte, 0)
// [5]      mas_initialized (1 byte, 0)
// [6]      mas_mode (1 byte, 0)
// [7..11]  stream_buffer_size (4 bytes uint32)
// [11..15] stream_buffer_fullness (4 bytes uint32)
// [15..19] bytes_received_H (4 bytes uint32)
// [19..23] bytes_received_L (4 bytes uint32)
// [23..25] signal_strength (2 bytes uint16, 0xFFFF)
// [25..29] jiffies (4 bytes uint32, client monotonic clock ms)
// [29..33] output_buffer_size (4 bytes uint32)
// [33..37] output_buffer_fullness (4 bytes uint32)
// [37..41] elapsed_seconds (4 bytes uint32)
// [41..43] voltage (2 bytes uint16, 0)
// [43..47] elapsed_milliseconds (4 bytes uint32)
// [47..51] server_timestamp (4 bytes uint32, reflected from LMS strm command)
// [51..53] error_code (2 bytes uint16, 0)
func EncodeStat(event [4]byte, streamBufSize uint32, streamBufFullness uint32, outBufSize uint32, outBufFullness uint32, bytesReceived uint64, jiffies uint32, elapsedMilliseconds uint32, serverTimestamp uint32) []byte {
	payloadLen := 53
	buf := make([]byte, 8+payloadLen)

	copy(buf[0:4], OpStat[:])
	binary.BigEndian.PutUint32(buf[4:8], uint32(payloadLen))

	offset := 8
	copy(buf[offset:offset+4], event[:]) // [0..4] event
	offset += 4

	buf[offset] = 0   // [4] num_crlf
	buf[offset+1] = 0 // [5] mas_initialized
	buf[offset+2] = 0 // [6] mas_mode
	offset += 3

	binary.BigEndian.PutUint32(buf[offset:offset+4], streamBufSize)       // [7..11] stream_buffer_size
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], streamBufFullness) // [11..15] stream_buffer_fullness
	offset += 8

	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(bytesReceived>>32)) // [15..19] bytes_received_H
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], uint32(bytesReceived))   // [19..23] bytes_received_L
	offset += 8

	binary.BigEndian.PutUint16(buf[offset:offset+2], 0xFFFF) // [23..25] signal_strength
	offset += 2

	binary.BigEndian.PutUint32(buf[offset:offset+4], jiffies) // [25..29] jiffies
	offset += 4

	binary.BigEndian.PutUint32(buf[offset:offset+4], outBufSize)                  // [29..33] output_buffer_size
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], outBufFullness)            // [33..37] output_buffer_fullness
	binary.BigEndian.PutUint32(buf[offset+8:offset+12], elapsedMilliseconds/1000) // [37..41] elapsed_seconds
	offset += 12

	binary.BigEndian.PutUint16(buf[offset:offset+2], 0) // [41..43] voltage
	offset += 2

	binary.BigEndian.PutUint32(buf[offset:offset+4], elapsedMilliseconds) // [43..47] elapsed_milliseconds
	binary.BigEndian.PutUint32(buf[offset+4:offset+8], serverTimestamp)   // [47..51] server_timestamp
	binary.BigEndian.PutUint16(buf[offset+8:offset+10], 0)                // [51..53] error_code

	return buf
}

// StrmCommand represents a parsed 'strm' command sent by LMS to control playback.
type StrmCommand struct {
	SubCommand       byte  // 's'=start, 'p'=pause, 'u'=unpause, 'q'=stop, 'f'=flush, 't'=tick, 'a'=skip ahead
	AutoStart        byte  // '0'=manual/sync, '1'=autostart, '2'=direct, '3'=wait for direct
	Format           byte  // 'p'=pcm, 'm'=mp3, 'f'=flac, 'a'=aac, 'o'=ogg/vorbis, 'u'=opus
	PCMSampleSize    byte  // '0'=8-bit, '1'=16-bit, '2'=24-bit, '3'=32-bit
	PCMSampleRate    byte  // '0'=11kHz, '1'=22kHz, '2'=32kHz, '3'=44.1kHz, '4'=48kHz, '5'=88.2kHz, '6'=96kHz
	PCMChannels      byte  // '1'=mono, '2'=stereo
	PCMEndianness    byte  // '0'=big, '1'=little
	Threshold        uint8 // buffer threshold KB
	SpdifEnable      uint8
	TransitionPeriod uint8
	TransitionType   uint8
	Flags            uint8
	OutputThreshold  uint8
	Slaves           uint8
	ReplayGain       uint32 // replay gain or target jiffies for unpause ('u')
	ServerPort       uint16 // streaming HTTP server port (e.g. 9000)
	ServerIP         net.IP // streaming server IP (0.0.0.0 if same as control connection)
	HTTPHeader       string // raw HTTP request string sent to streaming server
}

var ErrPacketTooShort = errors.New("slimproto packet too short")

// ParseStrm parses a raw 'strm' command payload received from LMS conforming to Squeezelite.
func ParseStrm(payload []byte) (*StrmCommand, error) {
	if len(payload) < 24 {
		return nil, ErrPacketTooShort
	}

	cmd := &StrmCommand{
		SubCommand:       payload[0],
		AutoStart:        payload[1],
		Format:           payload[2],
		PCMSampleSize:    payload[3],
		PCMSampleRate:    payload[4],
		PCMChannels:      payload[5],
		PCMEndianness:    payload[6],
		Threshold:        payload[7],
		SpdifEnable:      payload[8],
		TransitionPeriod: payload[9],
		TransitionType:   payload[10],
		Flags:            payload[11],
		OutputThreshold:  payload[12],
		Slaves:           payload[13],
		ReplayGain:       binary.BigEndian.Uint32(payload[14:18]),
		ServerPort:       binary.BigEndian.Uint16(payload[18:20]),
	}

	ipBytes := payload[20:24]
	cmd.ServerIP = net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])

	if len(payload) > 24 {
		cmd.HTTPHeader = string(payload[24:])
	}

	return cmd, nil
}
