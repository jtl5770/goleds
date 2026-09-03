package slimproto

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
)

// FLACDecoder decodes containerized or raw frame FLAC audio streams into 16-bit stereo PCM.
type FLACDecoder struct {
	pcmBuf []byte
}

// NewFLACDecoder creates a new FLACDecoder with a reusable conversion buffer.
func NewFLACDecoder() *FLACDecoder {
	return &FLACDecoder{
		pcmBuf: make([]byte, 16384),
	}
}

// Decode reads FLAC frames from r and streams converted 16-bit signed Little-Endian stereo PCM to out.
func (d *FLACDecoder) Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error {
	bufReader, ok := r.(*bufio.Reader)
	if !ok {
		bufReader = bufio.NewReader(r)
	}

	peek, _ := bufReader.Peek(4)
	hasFLACContainer := bytes.Equal(peek, []byte("fLaC"))

	sentThreshold := false
	chunkCount := 0

	if hasFLACContainer {
		stream, err := flac.New(bufReader)
		if err != nil {
			slog.Error("SlimProto FLAC container stream init failed", "error", err)
			return err
		}
		defer stream.Close()

		if stream.Info != nil {
			if cb != nil && stream.Info.SampleRate > 0 {
				cb.OnSampleRateChanged(stream.Info.SampleRate, uint32(stream.Info.NChannels))
			}
			slog.Info("SlimProto container FLAC initialized",
				"sampleRate", stream.Info.SampleRate,
				"channels", stream.Info.NChannels,
				"bitsPerSample", stream.Info.BitsPerSample)
		}

		for ctx.Err() == nil {
			f, err := stream.ParseNext()
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Info("SlimProto FLAC stream reached EOF", "framesDecoded", chunkCount)
				} else {
					slog.Debug("SlimProto FLAC parse stopped", "error", err, "framesDecoded", chunkCount)
				}
				break
			}
			if err := d.processFrame(f, out, thresholdBytes, &sentThreshold, cb); err != nil {
				return err
			}
			chunkCount++
		}
	} else {
		slog.Info("SlimProto decoding raw FLAC frame stream (LMS direct)")
		for ctx.Err() == nil {
			f, err := frame.Parse(bufReader)
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Info("SlimProto FLAC stream reached EOF", "framesDecoded", chunkCount)
				} else {
					slog.Debug("SlimProto raw FLAC parse stopped", "error", err, "framesDecoded", chunkCount)
				}
				break
			}
			if err := d.processFrame(f, out, thresholdBytes, &sentThreshold, cb); err != nil {
				return err
			}
			chunkCount++
		}
	}

	return ctx.Err()
}

func (d *FLACDecoder) processFrame(f *frame.Frame, out *AudioRingBuffer, thresholdBytes int, sentThreshold *bool, cb DecoderCallback) error {
	if len(f.Subframes) == 0 {
		return nil
	}
	nSamples := len(f.Subframes[0].Samples)
	if nSamples == 0 {
		return nil
	}

	nChannels := len(f.Subframes)
	bitsPerSample := int(f.BitsPerSample)
	sr := uint32(f.SampleRate)
	if cb != nil && sr > 0 {
		cb.OnSampleRateChanged(sr, uint32(nChannels))
	}

	// Convert FLAC subframe samples to 16-bit stereo LittleEndian PCM bytes (2 channels * 2 bytes = 4 bytes per frame)
	neededBytes := nSamples * 4
	if cap(d.pcmBuf) < neededBytes {
		d.pcmBuf = make([]byte, neededBytes)
	}
	pcm := d.pcmBuf[:neededBytes]

	idx := 0
	for i := 0; i < nSamples; i++ {
		for ch := 0; ch < 2; ch++ {
			var sample int32
			if nChannels == 1 {
				sample = f.Subframes[0].Samples[i]
			} else if ch < nChannels {
				sample = f.Subframes[ch].Samples[i]
			}
			if bitsPerSample > 16 {
				sample >>= (bitsPerSample - 16)
			} else if bitsPerSample < 16 && bitsPerSample > 0 {
				sample <<= (16 - bitsPerSample)
			}
			s16 := int16(sample)
			binary.LittleEndian.PutUint16(pcm[idx:], uint16(s16))
			idx += 2
		}
	}

	// Write to ring buffer (provides backpressure if full)
	if _, err := out.Write(pcm); err != nil {
		slog.Debug("SlimProto ring buffer write interrupted", "error", err)
		return err
	}

	// Signal threshold callback once enough buffer fullness is reached
	if !*sentThreshold && out.Available() >= thresholdBytes {
		*sentThreshold = true
		if cb != nil {
			cb.OnThresholdReached()
		}
	}

	return nil
}
