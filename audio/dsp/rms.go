package dsp

import (
	"encoding/binary"
	"math"
)

// CalculateLevels computes the Root Mean Square (RMS) and decibel (dB)
// levels across all frames in a 16-bit signed Little-Endian stereo PCM byte slice.
// Returns (leftDB, rightDB). Executes with zero heap allocations.
func CalculateLevels(pcm []byte) (leftDB, rightDB float64) {
	frames := len(pcm) / 4
	if frames == 0 {
		return -100, -100
	}

	var sumSqL, sumSqR float64
	for i := 0; i < frames; i++ {
		offset := i * 4
		sL := float64(int16(binary.LittleEndian.Uint16(pcm[offset : offset+2])))
		sR := float64(int16(binary.LittleEndian.Uint16(pcm[offset+2 : offset+4])))

		sumSqL += sL * sL
		sumSqR += sR * sR
	}

	rmsL := math.Sqrt(sumSqL / float64(frames))
	rmsR := math.Sqrt(sumSqR / float64(frames))

	// Normalize relative to full-scale 16-bit sine wave (32767.0). Clamped to -100 dB minimum.
	leftDB = 20 * math.Log10(math.Max(rmsL/32767.0, 1e-5))
	rightDB = 20 * math.Log10(math.Max(rmsR/32767.0, 1e-5))
	return leftDB, rightDB
}
