package slimproto

import (
	"context"
	"io"
)

// StreamFormat defines audio stream encodings supported by SlimProto.
type StreamFormat byte

const (
	FormatFLAC StreamFormat = 'f'
	FormatPCM  StreamFormat = 'p'
	FormatMP3  StreamFormat = 'm'
	FormatAAC  StreamFormat = 'a'
	FormatOGG  StreamFormat = 'o'
)

func (f StreamFormat) String() string {
	switch f {
	case FormatFLAC:
		return "FLAC"
	case FormatPCM:
		return "PCM"
	case FormatMP3:
		return "MP3"
	case FormatAAC:
		return "AAC"
	case FormatOGG:
		return "OGG"
	default:
		return string([]byte{byte(f)})
	}
}

// DecoderCallback notifies the audio pipeline of format and buffering milestones during streaming.
type DecoderCallback interface {
	// OnSampleRateChanged is invoked when stream audio parameters are extracted or updated.
	OnSampleRateChanged(sampleRate uint32, channels uint32)
	// OnThresholdReached is invoked once the ring buffer has accumulated enough bytes for threshold.
	OnThresholdReached()
}

// Decoder parses an encoded or raw audio stream from an io.Reader, converts samples
// to 16-bit signed stereo Little-Endian PCM, and pushes them into an AudioRingBuffer.
type Decoder interface {
	Decode(ctx context.Context, r io.Reader, out *AudioRingBuffer, thresholdBytes int, cb DecoderCallback) error
}
