package dsp

import (
	"encoding/binary"
	"math"
)

const (
	// MaxPCM16Amplitude represents the maximum positive amplitude of a 16-bit signed PCM sample.
	MaxPCM16Amplitude = 32767.0
	// SilenceFloorDB is the floor decibel level clamped for silence.
	SilenceFloorDB = -100.0
	// MinRMSClampingFloor prevents math.Log10 evaluation at or below zero.
	MinRMSClampingFloor = 1e-5
	// Stereo16BitFrameBytes is the byte size of one 16-bit stereo frame (2 channels * 2 bytes).
	Stereo16BitFrameBytes = 4
	// DecibelPowerMultiplier is the 20 * log10(x) multiplier for RMS amplitude.
	DecibelPowerMultiplier = 20.0
)

// CalculateLevels computes the Root Mean Square (RMS) and decibel (dB)
// levels across all frames in a 16-bit signed Little-Endian stereo PCM byte slice.
// Returns (leftDB, rightDB). Executes with zero heap allocations.
func CalculateLevels(pcm []byte) (leftDB, rightDB float64) {
	frames := len(pcm) / Stereo16BitFrameBytes
	if frames == 0 {
		return SilenceFloorDB, SilenceFloorDB
	}

	var sumSqL, sumSqR float64
	for i := 0; i < frames; i++ {
		offset := i * Stereo16BitFrameBytes
		sL := float64(int16(binary.LittleEndian.Uint16(pcm[offset : offset+2])))
		sR := float64(int16(binary.LittleEndian.Uint16(pcm[offset+2 : offset+4])))

		sumSqL += sL * sL
		sumSqR += sR * sR
	}

	rmsL := math.Sqrt(sumSqL / float64(frames))
	rmsR := math.Sqrt(sumSqR / float64(frames))

	// Normalize relative to full-scale 16-bit sine wave. Clamped to silence floor.
	leftDB = DecibelPowerMultiplier * math.Log10(math.Max(rmsL/MaxPCM16Amplitude, MinRMSClampingFloor))
	rightDB = DecibelPowerMultiplier * math.Log10(math.Max(rmsR/MaxPCM16Amplitude, MinRMSClampingFloor))
	return leftDB, rightDB
}
