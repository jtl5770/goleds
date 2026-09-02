package dsp

import (
	"encoding/binary"
	"math"
)

const (
	// MaxPCM16AmplitudeSquared represents (32767.0)^2 for normalizing power.
	MaxPCM16AmplitudeSquared = 32767.0 * 32767.0
	// SilenceFloorDB is the floor decibel level clamped for silence.
	SilenceFloorDB = -100.0
	// MinPowerClampingFloor prevents math.Log10 evaluation below -100 dB (1e-5 RMS = 1e-10 Power).
	MinPowerClampingFloor = 1e-10
	// Stereo16BitFrameBytes is the byte size of one 16-bit stereo frame (2 channels * 2 bytes).
	Stereo16BitFrameBytes = 4
	// DecibelPowerMultiplier is the 10 * log10(power) multiplier for RMS power.
	DecibelPowerMultiplier = 10.0
)

// CalculateLevels computes the Root Mean Square (RMS) decibel (dB) levels
// across all frames in a 16-bit signed Little-Endian stereo PCM byte slice.
// Returns (leftDB, rightDB). Executes with zero heap allocations and pure 64-bit integer registers.
func CalculateLevels(pcm []byte) (leftDB, rightDB float64) {
	frames := len(pcm) / Stereo16BitFrameBytes
	if frames == 0 {
		return SilenceFloorDB, SilenceFloorDB
	}

	var sumSqL, sumSqR uint64
	offset := 0
	end := frames * Stereo16BitFrameBytes

	// Unroll 4 frames (16 bytes = 2 uint64 loads) per iteration
	for offset <= end-16 {
		dw0 := binary.LittleEndian.Uint64(pcm[offset : offset+8])
		dw1 := binary.LittleEndian.Uint64(pcm[offset+8 : offset+16])

		sL0 := int64(int16(dw0))
		sR0 := int64(int16(dw0 >> 16))
		sL1 := int64(int16(dw0 >> 32))
		sR1 := int64(int16(dw0 >> 48))

		sL2 := int64(int16(dw1))
		sR2 := int64(int16(dw1 >> 16))
		sL3 := int64(int16(dw1 >> 32))
		sR3 := int64(int16(dw1 >> 48))

		sumSqL += uint64(sL0*sL0 + sL1*sL1 + sL2*sL2 + sL3*sL3)
		sumSqR += uint64(sR0*sR0 + sR1*sR1 + sR2*sR2 + sR3*sR3)

		offset += 16
	}

	// Handle trailing frames (1..3 frames)
	for offset <= end-4 {
		w := binary.LittleEndian.Uint32(pcm[offset : offset+4])
		sL := int64(int16(w))
		sR := int64(int16(w >> 16))
		sumSqL += uint64(sL * sL)
		sumSqR += uint64(sR * sR)
		offset += 4
	}

	// Power normalization: (sumSq / frames) / MaxPCM16Amplitude^2
	normFactor := float64(frames) * MaxPCM16AmplitudeSquared
	powerL := float64(sumSqL) / normFactor
	powerR := float64(sumSqR) / normFactor

	// dB = 10 * log10(power), clamped at silence floor (-100 dB)
	leftDB = DecibelPowerMultiplier * math.Log10(math.Max(powerL, MinPowerClampingFloor))
	rightDB = DecibelPowerMultiplier * math.Log10(math.Max(powerR, MinPowerClampingFloor))
	return leftDB, rightDB
}
