package slimproto

import (
	"context"
	"io"
)

// PCMDecoder ingests raw PCM byte streams directly into the ring buffer.
type PCMDecoder struct {
	buf []byte
}

// NewPCMDecoder creates a new PCMDecoder with an internal read buffer.
func NewPCMDecoder() *PCMDecoder {
	return &PCMDecoder{
		buf: make([]byte, 4096),
	}
}

// Decode reads raw PCM bytes from r and streams them into out.
func (d *PCMDecoder) Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error {
	sentThreshold := false

	for ctx.Err() == nil {
		n, err := r.Read(d.buf)
		if n > 0 {
			if _, werr := out.Write(d.buf[:n]); werr != nil {
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
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return ctx.Err()
}
