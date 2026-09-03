package slimproto

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/skrashevich/go-aac/pkg/adts"
	aacdec "github.com/skrashevich/go-aac/pkg/decoder"
)

var aacSampleRates = []uint32{
	96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350,
}

// AACDecoder decodes ADTS AAC audio streams into 16-bit signed stereo Little-Endian PCM.
type AACDecoder struct {
	rawBuf []byte
	pcmBuf []byte
}

// NewAACDecoder creates a new AACDecoder with preallocated conversion buffers.
func NewAACDecoder() *AACDecoder {
	return &AACDecoder{
		rawBuf: make([]byte, 32768),
		pcmBuf: make([]byte, 32768),
	}
}

// Decode reads ADTS AAC packets from r, decodes frames to 16-bit stereo Little-Endian PCM, and pushes to out.
func (d *AACDecoder) Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error {
	dec := aacdec.New()

	buf := d.rawBuf
	bufLen := 0
	initialized := false
	sentThreshold := false
	eof := false
	var header adts.Header

	for ctx.Err() == nil {
		// Read more data into buffer if room available and EOF not yet reached
		if !eof && bufLen < len(buf)/2 {
			n, err := r.Read(buf[bufLen:])
			if n > 0 {
				bufLen += n
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					eof = true
				} else {
					return err
				}
			}
		}

		if bufLen < 7 {
			if eof {
				break
			}
			continue
		}

		// Find ADTS syncword (0xFFF)
		syncIdx := -1
		for i := 0; i <= bufLen-2; i++ {
			if buf[i] == 0xFF && (buf[i+1]&0xF0) == 0xF0 {
				syncIdx = i
				break
			}
		}

		if syncIdx < 0 {
			// No syncword found in buffer, discard all but last byte
			if bufLen > 1 {
				buf[0] = buf[bufLen-1]
				bufLen = 1
			}
			if eof {
				break
			}
			continue
		}

		if syncIdx > 0 {
			copy(buf[0:], buf[syncIdx:bufLen])
			bufLen -= syncIdx
		}

		if bufLen < 7 {
			if eof {
				break
			}
			continue
		}

		hdr, err := adts.ReadHeaderFromBytes(buf[:bufLen])
		if err != nil || hdr.FrameLength < 7 || hdr.FrameLength > 8192 {
			// Corrupt ADTS header, advance 1 byte and search next syncword
			copy(buf[0:], buf[1:bufLen])
			bufLen--
			continue
		}

		frameLen := hdr.FrameLength
		if bufLen < frameLen {
			if eof {
				break // Incomplete trailing frame at EOF
			}
			continue // Need more bytes for complete frame
		}

		frameData := buf[:frameLen]

		if !initialized {
			header = hdr
			asc, aerr := adts.AudioSpecificConfig(hdr)
			if aerr != nil {
				slog.Error("SlimProto AAC ASC generation failed", "error", aerr)
				return fmt.Errorf("aac asc: %w", aerr)
			}
			if serr := dec.SetASC(asc[:]); serr != nil {
				slog.Error("SlimProto AAC decoder SetASC failed", "error", serr)
				return fmt.Errorf("aac set asc: %w", serr)
			}

			sr := uint32(44100)
			if hdr.SamplingIndex >= 0 && hdr.SamplingIndex < len(aacSampleRates) {
				sr = aacSampleRates[hdr.SamplingIndex]
			}
			if cb != nil && sr > 0 {
				cb.OnSampleRateChanged(sr, 2)
			}
			slog.Info("SlimProto AAC decoder initialized",
				"sampleRate", sr,
				"channels", hdr.ChannelConfig)
			initialized = true
		}

		// Decode the isolated ADTS frame (skipping ADTS header: 7 bytes or 9 bytes if CRC present)
		headerSize := 7
		if !header.ProtectionAbsent {
			headerSize = 9
		}
		if frameLen <= headerSize {
			copy(buf[0:], buf[frameLen:bufLen])
			bufLen -= frameLen
			continue
		}

		payload := frameData[headerSize:]
		samples, derr := dec.DecodeFrame(payload)

		// Advance input buffer by frameLen
		copy(buf[0:], buf[frameLen:bufLen])
		bufLen -= frameLen

		if derr != nil {
			slog.Debug("SlimProto AAC frame decode error", "error", derr)
			continue
		}

		// Process output PCM float32 samples to 16-bit signed stereo Little-Endian
		if len(samples) > 0 {
			ch := header.ChannelConfig
			neededBytes := len(samples) * 2
			if ch == 1 {
				neededBytes = len(samples) * 4 // Mono duplicate to stereo
			}
			if cap(d.pcmBuf) < neededBytes {
				d.pcmBuf = make([]byte, neededBytes)
			}
			outPcm := d.pcmBuf[:neededBytes]

			outIdx := 0
			if ch == 1 {
				for _, s := range samples {
					val := int16(math.Max(-32768, math.Min(32767, float64(s*32767.0))))
					binary.LittleEndian.PutUint16(outPcm[outIdx:], uint16(val))
					binary.LittleEndian.PutUint16(outPcm[outIdx+2:], uint16(val))
					outIdx += 4
				}
			} else {
				for _, s := range samples {
					val := int16(math.Max(-32768, math.Min(32767, float64(s*32767.0))))
					binary.LittleEndian.PutUint16(outPcm[outIdx:], uint16(val))
					outIdx += 2
				}
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
		}
	}

	return ctx.Err()
}
