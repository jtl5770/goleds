package slimproto

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/hajimehoshi/go-mp3"
)

// MP3Decoder decodes MPEG-1/2/2.5 Layer III audio streams into 16-bit signed stereo Little-Endian PCM.
type MP3Decoder struct {
	pcmBuf []byte
}

// NewMP3Decoder creates a new MP3Decoder with a preallocated read buffer.
func NewMP3Decoder() *MP3Decoder {
	return &MP3Decoder{
		pcmBuf: make([]byte, 8192),
	}
}

// Decode parses the MP3 stream from r, converts frames to 16-bit stereo Little-Endian PCM, and pushes to out.
func (d *MP3Decoder) Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error {
	dec, err := mp3.NewDecoder(r)
	if err != nil {
		slog.Error("SlimProto MP3 decoder init failed", "error", err)
		return fmt.Errorf("mp3 init: %w", err)
	}

	sr := uint32(dec.SampleRate())
	if cb != nil && sr > 0 {
		cb.OnSampleRateChanged(sr, 2)
	}
	slog.Info("SlimProto MP3 decoder initialized", "sampleRate", sr)

	sentThreshold := false

	for ctx.Err() == nil {
		n, err := dec.Read(d.pcmBuf)
		if n > 0 {
			if _, werr := out.Write(d.pcmBuf[:n]); werr != nil {
				return werr
			}

			if !sentThreshold && out.Available() >= thresholdBytes {
				sentThreshold = true
				if cb != nil {
					cb.OnThresholdReached()
				}
			}
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
