package hardware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

// Mocking the rpio library

type mockSPI struct {
	exchangeFunc func(write []byte) []byte
}

func (m *mockSPI) Exchange(write []byte) []byte {
	return m.exchangeFunc(write)
}

func TestReadAdc(t *testing.T) {
	oldConfig := c.CONFIG
	t.Cleanup(func() { c.CONFIG = oldConfig })
	c.CONFIG.RealHW = false
	c.CONFIG.Hardware.SpiMultiplexGPIO = map[string]struct {
		Low  []int `yaml:"Low"`
		High []int `yaml:"High"`
	}{
		"test": {Low: []int{1}, High: []int{2}},
	}
	spimultiplexcfg = make(map[string]gpiocfg)
	spimultiplexcfg["test"] = gpiocfg{}

	spi = &mockSPI{
		exchangeFunc: func(write []byte) []byte {
			assert.Equal(t, []byte{1, (8 + 3) << 4, 0}, write)
			return []byte{0, 0b11, 0b11111111}
		},
	}

	val := ReadAdc("test", 3)
	assert.Equal(t, 1023, val, "ADC value should be 1023")
}

func TestSetLedSegment_ws2801(t *testing.T) {
	oldConfig := c.CONFIG
	t.Cleanup(func() { c.CONFIG = oldConfig })
	c.CONFIG.RealHW = false
	c.CONFIG.Hardware.LEDType = "ws2801"
	c.CONFIG.Hardware.Display.ColorCorrection = []float64{1, 1, 1}
	c.CONFIG.Hardware.SpiMultiplexGPIO = map[string]struct {
		Low  []int `yaml:"Low"`
		High []int `yaml:"High"`
	}{
		"test": {Low: []int{1}, High: []int{2}},
	}
	spimultiplexcfg = make(map[string]gpiocfg)
	spimultiplexcfg["test"] = gpiocfg{}

	var capturedDisplay []byte
	spi = &mockSPI{
		exchangeFunc: func(write []byte) []byte {
			capturedDisplay = write
			return write
		},
	}

	leds := []p.Led{
		{Red: 10, Green: 20, Blue: 30},
		{Red: 40, Green: 50, Blue: 60},
	}

	SetLedSegment("test", leds)

	expectedDisplay := []byte{10, 20, 30, 40, 50, 60}
	assert.Equal(t, expectedDisplay, capturedDisplay, "ws2801 display data should be correct")
}

func TestSetLedSegment_apa102(t *testing.T) {
	oldConfig := c.CONFIG
	t.Cleanup(func() { c.CONFIG = oldConfig })
	c.CONFIG.RealHW = false
	c.CONFIG.Hardware.LEDType = "apa102"
	c.CONFIG.Hardware.Display.ColorCorrection = []float64{1, 1, 1}
	c.CONFIG.Hardware.Display.APA102_Brightness = 31
	c.CONFIG.Hardware.SpiMultiplexGPIO = map[string]struct {
		Low  []int `yaml:"Low"`
		High []int `yaml:"High"`
	}{
		"test": {Low: []int{1}, High: []int{2}},
	}
	spimultiplexcfg = make(map[string]gpiocfg)
	spimultiplexcfg["test"] = gpiocfg{}

	var capturedDisplay []byte
	spi = &mockSPI{
		exchangeFunc: func(write []byte) []byte {
			capturedDisplay = write
			return write
		},
	}

	leds := []p.Led{
		{Red: 10, Green: 20, Blue: 30},
		{Red: 40, Green: 50, Blue: 60},
	}

	SetLedSegment("test", leds)

	// Frame start + 2 LEDs + Frame end
	expectedDisplay := []byte{
		0x00, 0x00, 0x00, 0x00, // Start frame
		0xE0 | 31, 30, 20, 10, // LED 1
		0xE0 | 31, 60, 50, 40, // LED 2
		0xFF,                   // End frame
	}
	assert.Equal(t, expectedDisplay, capturedDisplay, "apa102 display data should be correct")
}
