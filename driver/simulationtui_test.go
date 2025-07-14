package driver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	p "lautenbacher.net/goleds/producer"
)

func TestScaledColor(t *testing.T) {
	// Test case 1: Basic scaling with rounding
	led := p.Led{Red: 50, Green: 100, Blue: 200}
	// Blue is max (200), factor is 255/200 = 1.275
	// R = 50 * 1.275 = 63.75 -> round -> 64 (0x40)
	// G = 100 * 1.275 = 127.5 -> round -> 128 (0x80)
	// B = 200 * 1.275 = 255 -> round -> 255 (0xff)
	expected := "[#4080ff]"
	assert.Equal(t, expected, scaledColor(led))

	// Test case 2: A value is already 255
	led = p.Led{Red: 255, Green: 10, Blue: 100}
	// Factor is 255/255 = 1
	expected = "[#ff0a64]"
	assert.Equal(t, expected, scaledColor(led))

	// Test case 3: All values are zero
	led = p.Led{Red: 0, Green: 0, Blue: 0}
	expected = "[#000000]"
	assert.Equal(t, expected, scaledColor(led))

	// Test case 4: Floating point values with rounding
	led = p.Led{Red: 25.5, Green: 128.1, Blue: 60.9}
	// Green is max (128.1), factor is 255/128.1 = 1.9906...
	// R = 25.5 * 1.99... = 50.76... -> round -> 51 (0x33)
	// G = 128.1 * 1.99... = 255 -> round -> 255 (0xff)
	// B = 60.9 * 1.99... = 121.22... -> round -> 121 (0x79)
	expected = "[#33ff79]"
	assert.Equal(t, expected, scaledColor(led))
}

func TestSimulateLed(t *testing.T) {
	ledsTotal := 20

	// Test case 1: Non-visible segment
	segNonVisible := NewLedSegment(5, 10, "spi1", false, false, ledsTotal)
	top, bot := simulateLed(segNonVisible)
	// Length is 10 - 5 + 1 = 6
	assert.Equal(t, "      ", top)
	assert.Equal(t, "······", bot)

	// Test case 2: Visible segment
	segVisible := NewLedSegment(0, 3, "spi1", false, true, ledsTotal) // 4 LEDs long
	segVisible.leds = []p.Led{
		{Red: 255, Green: 0, Blue: 0},
		{Red: 0, Green: 255, Blue: 0},
		{Red: 0, Green: 0, Blue: 255},
		{Red: 100, Green: 100, Blue: 100},
	}

	top, bot = simulateLed(segVisible)

	expectedTop := "[#ff0000]█[-][#00ff00]█[-][#0000ff]█[-][#ffffff]█[-]"
	expectedBot := "[#ff0000]█[-][#00ff00]█[-][#0000ff]█[-][#ffffff]█[-]"

	// Normalize by removing color codes for simple comparison if needed, but direct is better
	assert.Equal(t, expectedTop, top)
	assert.Equal(t, expectedBot, bot)
	assert.Equal(t, len(segVisible.leds), strings.Count(top, "█"))
	assert.Equal(t, len(segVisible.leds), strings.Count(bot, "█"))
}
