package producer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	c "lautenbacher.net/goleds/config"
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
	c.CONFIG.Hardware.Display.LedsTotal = 5

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

	sumLeds := CombineLeds(ledRanges)

	assert.Len(t, sumLeds, 5)

	// Check combined values
	assert.Equal(t, float64(10), sumLeds[0].Red)
	assert.Equal(t, float64(0), sumLeds[0].Green)
	assert.Equal(t, float64(20), sumLeds[0].Blue)

	assert.Equal(t, float64(5), sumLeds[1].Red)
	assert.Equal(t, float64(10), sumLeds[1].Green)
	assert.Equal(t, float64(5), sumLeds[1].Blue)

	// Check that the rest are zero
	assert.True(t, sumLeds[2].IsEmpty())
	assert.True(t, sumLeds[3].IsEmpty())
	assert.True(t, sumLeds[4].IsEmpty())
}
