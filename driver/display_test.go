package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

func TestNewLedSegment(t *testing.T) {
	oldConfig := c.CONFIG
	t.Cleanup(func() { c.CONFIG = oldConfig })
	c.CONFIG.Hardware.Display.LedsTotal = 100

	// Test normal creation
	seg := NewLedSegment(10, 20, "spi1", false, true)
	assert.Equal(t, 10, seg.firstled)
	assert.Equal(t, 20, seg.lastled)
	assert.Equal(t, "spi1", seg.spimultiplex)
	assert.False(t, seg.reverse)
	assert.True(t, seg.visible)

	// Test reversed first/last
	seg = NewLedSegment(20, 10, "spi1", false, true)
	assert.Equal(t, 10, seg.firstled)
	assert.Equal(t, 20, seg.lastled)

	// Test clamping
	seg = NewLedSegment(-10, 110, "spi1", false, true)
	assert.Equal(t, 0, seg.firstled)
	assert.Equal(t, 99, seg.lastled)

	// Test non-visible
	seg = NewLedSegment(10, 20, "spi1", false, false)
	assert.Equal(t, "__", seg.spimultiplex)
}

func TestLedSegmentGetAndSet(t *testing.T) {
	oldConfig := c.CONFIG
	t.Cleanup(func() { c.CONFIG = oldConfig })
	c.CONFIG.Hardware.Display.LedsTotal = 10
	seg := NewLedSegment(2, 5, "spi1", false, true)

	sumleds := make([]p.Led, 10)
	for i := range sumleds {
		sumleds[i] = p.Led{Red: float64(i)}
	}

	seg.setSegmentLeds(sumleds)
	leds := seg.getSegmentLeds()

	assert.Len(t, leds, 4)
	assert.Equal(t, float64(2), leds[0].Red)
	assert.Equal(t, float64(5), leds[3].Red)

	// Test reverse
	seg = NewLedSegment(2, 5, "spi1", true, true)
	seg.setSegmentLeds(sumleds)
	leds = seg.getSegmentLeds()

	assert.Len(t, leds, 4)
	assert.Equal(t, float64(5), leds[0].Red)
	assert.Equal(t, float64(2), leds[3].Red)
}

func TestInitDisplay(t *testing.T) {
	oldConfig := c.CONFIG
	t.Cleanup(func() { c.CONFIG = oldConfig })
	c.CONFIG.Hardware.Display.LedsTotal = 10
	c.CONFIG.Hardware.Display.LedSegments = map[string][]struct {
		FirstLed     int    `yaml:"FirstLed"`
		LastLed      int    `yaml:"LastLed"`
		SpiMultiplex string `yaml:"SpiMultiplex"`
		Reverse      bool   `yaml:"Reverse"`
	}{
		"test": {
			{FirstLed: 0, LastLed: 3, SpiMultiplex: "spi1", Reverse: false},
			{FirstLed: 8, LastLed: 9, SpiMultiplex: "spi2", Reverse: true},
		},
	}

	InitDisplay()

	assert.NotNil(t, SEGMENTS)
	assert.Len(t, SEGMENTS["test"], 3) // 2 visible, 1 non-visible gap

	// Check visible segments
	assert.Equal(t, 0, SEGMENTS["test"][0].firstled)
	assert.Equal(t, 3, SEGMENTS["test"][0].lastled)
	assert.Equal(t, "spi1", SEGMENTS["test"][0].spimultiplex)

	assert.Equal(t, 8, SEGMENTS["test"][2].firstled)
	assert.Equal(t, 9, SEGMENTS["test"][2].lastled)
	assert.Equal(t, "spi2", SEGMENTS["test"][2].spimultiplex)
	assert.True(t, SEGMENTS["test"][2].reverse)

	// Check non-visible segments (gaps)
	assert.Equal(t, 4, SEGMENTS["test"][1].firstled)
	assert.Equal(t, 7, SEGMENTS["test"][1].lastled)
	assert.False(t, SEGMENTS["test"][1].visible)

	// Test overlapping segments panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	c.CONFIG.Hardware.Display.LedSegments["test"] = append(c.CONFIG.Hardware.Display.LedSegments["test"], struct {
		FirstLed     int    `yaml:"FirstLed"`
		LastLed      int    `yaml:"LastLed"`
		SpiMultiplex string `yaml:"SpiMultiplex"`
		Reverse      bool   `yaml:"Reverse"`
	}{FirstLed: 3, LastLed: 5, SpiMultiplex: "spi3", Reverse: false})
	InitDisplay()
}
