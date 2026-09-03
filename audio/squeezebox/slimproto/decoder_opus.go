package slimproto

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/pion/opus"
	"github.com/pion/opus/pkg/oggreader"
)

// OpusDecoder decodes Ogg Opus audio streams into 16-bit signed stereo Little-Endian PCM.
type OpusDecoder struct {
	sampleBuf []int16
	pcmBuf    []byte
}

// NewOpusDecoder creates a new OpusDecoder with preallocated conversion buffers.
func NewOpusDecoder() *OpusDecoder {
	return &OpusDecoder{
		sampleBuf: make([]int16, 5760*2), // Max Opus frame size is 120ms @ 48kHz = 5760 samples per channel
		pcmBuf:    make([]byte, 5760*4),
	}
}

// Decode reads Ogg Opus audio from r, decodes frames to 16-bit stereo Little-Endian PCM, and pushes to out.
func (d *OpusDecoder) Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error {
	ogg, header, err := oggreader.NewWith(r)
	if err != nil {
		slog.Error("SlimProto Opus decoder init failed", "error", err)
		return fmt.Errorf("opus ogg init: %w", err)
	}

	dec, err := opus.NewDecoderWithOutput(48000, 2)
	if err != nil {
		slog.Error("SlimProto Opus decoder creation failed", "error", err)
		return fmt.Errorf("opus decoder create: %w", err)
	}

	if cb != nil {
		cb.OnSampleRateChanged(48000, 2)
	}

	slog.Info("SlimProto Opus decoder initialized",
		"sampleRate", 48000,
		"channels", header.Channels)

	sentThreshold := false

	for ctx.Err() == nil {
		payload, _, perr := ogg.ParseNextPacket()
		if perr != nil {
			if errors.Is(perr, io.EOF) {
				slog.Info("SlimProto Opus stream reached EOF")
				break
			}
			slog.Debug("SlimProto Opus packet parse error", "error", perr)
			return perr
		}

		samplesPerChannel, derr := dec.DecodeToInt16(payload, d.sampleBuf)
		if derr != nil {
			slog.Debug("SlimProto Opus frame decode error", "error", derr)
			continue
		}

		if samplesPerChannel > 0 {
			totalSamples := samplesPerChannel * 2
			totalBytes := totalSamples * 2
			if cap(d.pcmBuf) < totalBytes {
				d.pcmBuf = make([]byte, totalBytes)
			}
			outPcm := d.pcmBuf[:totalBytes]

			for i := 0; i < totalSamples; i++ {
				binary.LittleEndian.PutUint16(outPcm[i*2:], uint16(d.sampleBuf[i]))
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
