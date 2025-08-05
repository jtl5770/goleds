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

func TestLed_Max(t *testing.T) {
	led1 := Led{Red: 10, Green: 20, Blue: 30}
	led2 := Led{Red: 5, Green: 25, Blue: 15}

	maxLed := led1.Max(led2)

	assert.Equal(t, float64(10), maxLed.Red)
	assert.Equal(t, float64(25), maxLed.Green)
	assert.Equal(t, float64(30), maxLed.Blue)
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
