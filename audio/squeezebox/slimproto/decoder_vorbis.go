package slimproto

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/jfreymuth/oggvorbis"
)

// VorbisDecoder decodes Ogg Vorbis audio streams into 16-bit signed stereo Little-Endian PCM.
type VorbisDecoder struct {
	f32Buf []float32
	pcmBuf []byte
}

// NewVorbisDecoder creates a new VorbisDecoder with preallocated conversion buffers.
func NewVorbisDecoder() *VorbisDecoder {
	return &VorbisDecoder{
		f32Buf: make([]float32, 4096),
		pcmBuf: make([]byte, 16384),
	}
}

// Decode reads Ogg Vorbis audio from r, decodes frames to 16-bit stereo Little-Endian PCM, and pushes to out.
func (d *VorbisDecoder) Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error {
	ovr, err := oggvorbis.NewReader(r)
	if err != nil {
		slog.Error("SlimProto Vorbis decoder init failed", "error", err)
		return fmt.Errorf("vorbis init: %w", err)
	}

	sr := uint32(ovr.SampleRate())
	channels := ovr.Channels()
	if cb != nil && sr > 0 {
		cb.OnSampleRateChanged(sr, 2)
	}

	slog.Info("SlimProto Vorbis decoder initialized",
		"sampleRate", sr,
		"channels", channels)

	sentThreshold := false

	for ctx.Err() == nil {
		n, rerr := ovr.Read(d.f32Buf)
		if n > 0 {
			var neededBytes int
			if channels == 1 {
				neededBytes = n * 4 // Mono: 1 float32 sample becomes 2 channels * 2 bytes = 4 bytes
			} else {
				neededBytes = (n / channels) * 4 // Stereo / multichannel: take first 2 channels
			}

			if cap(d.pcmBuf) < neededBytes {
				d.pcmBuf = make([]byte, neededBytes)
			}
			outPcm := d.pcmBuf[:neededBytes]

			outIdx := 0
			if channels == 1 {
				for i := 0; i < n; i++ {
					val := int16(math.Max(-32768, math.Min(32767, float64(d.f32Buf[i]*32767.0))))
					binary.LittleEndian.PutUint16(outPcm[outIdx:], uint16(val))
					binary.LittleEndian.PutUint16(outPcm[outIdx+2:], uint16(val))
					outIdx += 4
				}
			} else {
				for i := 0; i+channels <= n; i += channels {
					leftVal := int16(math.Max(-32768, math.Min(32767, float64(d.f32Buf[i]*32767.0))))
					rightVal := int16(math.Max(-32768, math.Min(32767, float64(d.f32Buf[i+1]*32767.0))))
					binary.LittleEndian.PutUint16(outPcm[outIdx:], uint16(leftVal))
					binary.LittleEndian.PutUint16(outPcm[outIdx+2:], uint16(rightVal))
					outIdx += 4
				}
			}

			if _, werr := out.Write(outPcm[:outIdx]); werr != nil {
				return werr
			}

			if !sentThreshold && out.Available() >= thresholdBytes {
				sentThreshold = true
				if cb != nil {
					cb.OnThresholdReached()
				}
			}
		}

		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				slog.Info("SlimProto Vorbis stream reached EOF")
				break
			}
			slog.Debug("SlimProto Vorbis read error", "error", rerr)
			return rerr
		}
	}

	return ctx.Err()
}
