package producer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLed_IsEmpty(t *testing.T) {
	led := Led{Red: 0, Green: 0, Blue: 0}
	assert.True(t, led.IsEmpty(), "IsEmpty should be true for a zero Led")

	led = Led{Red: 1, Green: 0, Blue: 0}
	assert.False(t, led.IsEmpty(), "IsEmpty should be false for a non-zero Led")
}

func TestCombineLeds(t *testing.T) {
	ledsTotal := 5
	combinedLeds := make([]Led, ledsTotal)
	ledRanges := map[string][]Led{
		"range1": {
			{Red: 10, Green: 0, Blue: 0},
			{Red: 0, Green: 10, Blue: 0},
		},
		"range2": {
			{Red: 0, Green: 0, Blue: 20},
			{Red: 5, Green: 5, Blue: 5},
		},
	}

	CombineLeds(ledRanges, combinedLeds)

	assert.Len(t, combinedLeds, 5)

	// Check combined values
	assert.Equal(t, float64(10), combinedLeds[0].Red)
	assert.Equal(t, float64(0), combinedLeds[0].Green)
	assert.Equal(t, float64(20), combinedLeds[0].Blue)

	assert.Equal(t, float64(5), combinedLeds[1].Red)
	assert.Equal(t, float64(10), combinedLeds[1].Green)
	assert.Equal(t, float64(5), combinedLeds[1].Blue)

	// Check that the rest are zero
	assert.True(t, combinedLeds[2].IsEmpty())
	assert.True(t, combinedLeds[3].IsEmpty())
	assert.True(t, combinedLeds[4].IsEmpty())
}

func BenchmarkCombineLeds(b *testing.B) {
	const ledsTotal = 300
	target := make([]Led, ledsTotal)
	ledRanges := map[string][]Led{
		"night": make([]Led, ledsTotal),
		"audio": make([]Led, ledsTotal),
		"blob":  make([]Led, ledsTotal),
	}
	for i := 0; i < ledsTotal; i++ {
		ledRanges["night"][i] = Led{Red: 10, Green: 5, Blue: 2}
		ledRanges["audio"][i] = Led{Red: float64(i % 100), Green: float64((i * 2) % 100), Blue: 10}
		ledRanges["blob"][i] = Led{Red: 0, Green: float64(i % 50), Blue: float64(i % 200)}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CombineLeds(ledRanges, target)
	}
}
