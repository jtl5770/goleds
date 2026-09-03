package slimproto

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
)

// PCMConfig defines the stream parameters for incoming raw PCM audio.
type PCMConfig struct {
	SampleRate    uint32
	BitsPerSample int
	Channels      int
	LittleEndian  bool
}

// DefaultPCMConfig returns standard CD-quality 44.1kHz 16-bit stereo Big-Endian configuration (LMS default).
func DefaultPCMConfig() PCMConfig {
	return PCMConfig{
		SampleRate:    44100,
		BitsPerSample: 16,
		Channels:      2,
		LittleEndian:  false, // LMS sends Big-Endian by default for PCM
	}
}

// ParsePCMConfig parses SlimProto strm command byte codes into a PCMConfig.
func ParsePCMConfig(sampleRateCode, sampleSizeCode, channelsCode, endianCode byte) PCMConfig {
	cfg := DefaultPCMConfig()

	// Sample Rate mapping according to SlimProto specification
	switch sampleRateCode {
	case '0':
		cfg.SampleRate = 11025
	case '1':
		cfg.SampleRate = 22050
	case '2':
		cfg.SampleRate = 32000
	case '3':
		cfg.SampleRate = 44100
	case '4':
		cfg.SampleRate = 48000
	case '5':
		cfg.SampleRate = 88200
	case '6':
		cfg.SampleRate = 96000
	case '7':
		cfg.SampleRate = 176400
	case '8':
		cfg.SampleRate = 192000
	case '9':
		cfg.SampleRate = 352800
	case '?':
		cfg.SampleRate = 384000
	default:
		cfg.SampleRate = 44100
	}

	// Bit depth
	switch sampleSizeCode {
	case '0':
		cfg.BitsPerSample = 8
	case '1':
		cfg.BitsPerSample = 16
	case '2':
		cfg.BitsPerSample = 24
	case '3':
		cfg.BitsPerSample = 32
	default:
		cfg.BitsPerSample = 16
	}

	// Channels
	if channelsCode == '1' {
		cfg.Channels = 1
	} else {
		cfg.Channels = 2
	}

	// Endianness: '0' = Big-Endian, '1' = Little-Endian
	cfg.LittleEndian = (endianCode == '1')

	return cfg
}

// PCMDecoder ingests and normalizes raw PCM audio streams (converting to 16-bit signed stereo Little-Endian).
type PCMDecoder struct {
	config PCMConfig
	rawBuf []byte
	pcmBuf []byte
}

// NewPCMDecoder creates a new PCMDecoder with preallocated conversion buffers.
func NewPCMDecoder(configs ...PCMConfig) *PCMDecoder {
	cfg := DefaultPCMConfig()
	if len(configs) > 0 {
		cfg = configs[0]
	}
	return &PCMDecoder{
		config: cfg,
		rawBuf: make([]byte, 8192),
		pcmBuf: make([]byte, 16384),
	}
}

// Decode reads raw PCM bytes from r, converts samples to 16-bit signed stereo Little-Endian PCM, and streams to out.
func (d *PCMDecoder) Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error {
	cfg := d.config
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 44100
	}
	if cfg.BitsPerSample == 0 {
		cfg.BitsPerSample = 16
	}
	if cfg.Channels == 0 {
		cfg.Channels = 2
	}

	if cb != nil && cfg.SampleRate > 0 {
		cb.OnSampleRateChanged(cfg.SampleRate, 2)
	}

	bytesPerSample := cfg.BitsPerSample / 8
	bytesPerFrame := cfg.Channels * bytesPerSample
	if bytesPerFrame <= 0 {
		bytesPerFrame = 4
	}

	sentThreshold := false
	leftover := 0

	for ctx.Err() == nil {
		// Read into rawBuf after any leftover unaligned bytes from previous read
		n, err := r.Read(d.rawBuf[leftover:])
		totalBytes := leftover + n
		leftover = 0

		if totalBytes >= bytesPerFrame {
			framesToProcess := totalBytes / bytesPerFrame
			consumedBytes := framesToProcess * bytesPerFrame
			neededOutBytes := framesToProcess * 4 // 16-bit stereo = 4 bytes per frame

			if cap(d.pcmBuf) < neededOutBytes {
				d.pcmBuf = make([]byte, neededOutBytes)
			}
			outPcm := d.pcmBuf[:neededOutBytes]

			inIdx := 0
			outIdx := 0

			for f := 0; f < framesToProcess; f++ {
				var left, right int16

				// Extract Left channel (or Mono)
				left = extractSample(d.rawBuf[inIdx:inIdx+bytesPerSample], cfg.BitsPerSample, cfg.LittleEndian)
				inIdx += bytesPerSample

				if cfg.Channels == 1 {
					right = left // Mono duplicate to stereo
				} else {
					right = extractSample(d.rawBuf[inIdx:inIdx+bytesPerSample], cfg.BitsPerSample, cfg.LittleEndian)
					inIdx += bytesPerSample
				}

				binary.LittleEndian.PutUint16(outPcm[outIdx:], uint16(left))
				binary.LittleEndian.PutUint16(outPcm[outIdx+2:], uint16(right))
				outIdx += 4
			}

			if _, werr := out.Write(outPcm); werr != nil {
				return werr
			}

			if !sentThreshold && out.Available() >= thresholdBytes {
				sentThreshold = true
				if cb != nil {
					cb.OnThresholdReached()
				}
			}

			// Save unaligned leftover bytes
			remaining := totalBytes - consumedBytes
			if remaining > 0 {
				copy(d.rawBuf[0:remaining], d.rawBuf[consumedBytes:totalBytes])
				leftover = remaining
			}
		} else if totalBytes > 0 {
			leftover = totalBytes
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}

	return ctx.Err()
}

// extractSample converts a single channel raw byte slice of 8/16/24/32 bits to a signed 16-bit integer.
func extractSample(b []byte, bits int, littleEndian bool) int16 {
	switch bits {
	case 8:
		// 8-bit unsigned PCM to signed 16-bit
		return int16(int(b[0])-128) << 8
	case 16:
		if littleEndian {
			return int16(binary.LittleEndian.Uint16(b))
		}
		return int16(binary.BigEndian.Uint16(b))
	case 24:
		var val int32
		if littleEndian {
			val = (int32(b[2]) << 24) | (int32(b[1]) << 16) | (int32(b[0]) << 8)
		} else {
			val = (int32(b[0]) << 24) | (int32(b[1]) << 16) | (int32(b[2]) << 8)
		}
		return int16(val >> 16)
	case 32:
		var val int32
		if littleEndian {
			val = int32(binary.LittleEndian.Uint32(b))
		} else {
			val = int32(binary.BigEndian.Uint32(b))
		}
		return int16(val >> 16)
	default:
		return 0
	}
}
