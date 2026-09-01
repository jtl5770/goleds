package dsp

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestCalculateLevels_Silence(t *testing.T) {
	pcm := make([]byte, 100*4) // 100 frames of zeros

	leftDB, rightDB := CalculateLevels(pcm)
	if leftDB != -100 || rightDB != -100 {
		t.Errorf("Expected silence to be -100 dB, got Left=%.2f, Right=%.2f", leftDB, rightDB)
	}
}

func TestCalculateLevels_SineWave(t *testing.T) {
	frames := 1000
	pcm := make([]byte, frames*4)

	// Generate full-scale 1 kHz sine wave at 44.1 kHz on Left channel, silence on Right
	for i := 0; i < frames; i++ {
		tVal := float64(i) / 44100.0
		sampleL := int16(32767.0 * math.Sin(2*math.Pi*1000.0*tVal))
		sampleR := int16(0)

		binary.LittleEndian.PutUint16(pcm[i*4:], uint16(sampleL))
		binary.LittleEndian.PutUint16(pcm[i*4+2:], uint16(sampleR))
	}

	leftDB, rightDB := CalculateLevels(pcm)

	// Full-scale sine wave RMS is 1/sqrt(2) ≈ 0.7071 -> 20*log10(0.7071) ≈ -3.01 dB
	if math.Abs(leftDB-(-3.01)) > 0.1 {
		t.Errorf("Expected Left channel sine wave ≈ -3.01 dB, got %.2f dB", leftDB)
	}
	if rightDB != -100 {
		t.Errorf("Expected Right channel silence = -100 dB, got %.2f dB", rightDB)
	}
}

func BenchmarkCalculateLevels(b *testing.B) {
	frames := 1323 // 30ms @ 44.1kHz
	pcm := make([]byte, frames*4)

	// Pre-fill with sine wave samples
	for i := 0; i < frames; i++ {
		binary.LittleEndian.PutUint16(pcm[i*4:], uint16(16000))
		binary.LittleEndian.PutUint16(pcm[i*4+2:], uint16(16000))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CalculateLevels(pcm)
	}
}
