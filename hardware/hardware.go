package hardware

import (
	"log"
	"math"
	"sync"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

const (
	LEDType = "ws2801"
)

var (
	spiMutex        sync.Mutex
	spimultiplexcfg map[string]gpiocfg
)

type gpiocfg struct {
	low  []rpio.Pin
	high []rpio.Pin
}

var spi SPI

// InitHardware initializes the hardware. This includes the SPI and the GPIO
// multiplexer. If the configuration is set to use real hardware this function
// will panic if the hardware cannot be initialized. If the configuration is
// set to use fake hardware this function will print a log message.
func InitHardware() {
	if c.CONFIG.RealHW {
		log.Println("Initialise GPI and Spi...")
		if err := rpio.Open(); err != nil {
			panic(err)
		}
		if err := rpio.SpiBegin(rpio.Spi0); err != nil {
			panic(err)
		}

		rpio.SpiSpeed(c.CONFIG.Hardware.SPIFrequency)
		spi = &rpioSPI{}

		spimultiplexcfg = make(map[string]gpiocfg, len(c.CONFIG.Hardware.SpiMultiplexGPIO))

		for key, cfg := range c.CONFIG.Hardware.SpiMultiplexGPIO {
			low := make([]rpio.Pin, 0, len(cfg.Low))
			high := make([]rpio.Pin, 0, len(cfg.High))
			for _, pin := range cfg.Low {
				rpiopin := rpio.Pin(pin)
				rpiopin.Output()
				low = append(low, rpiopin)
			}
			for _, pin := range cfg.High {
				rpiopin := rpio.Pin(pin)
				rpiopin.Output()
				high = append(high, rpiopin)
			}
			spimultiplexcfg[key] = gpiocfg{
				low:  low,
				high: high,
			}
		}
	} else {
		log.Println("No GPI init done as we are not running on real hardware...")
	}
}

type rpioSPI struct{}

func (s *rpioSPI) Exchange(write []byte) []byte {
	rpio.SpiExchange(write)
	return write
}

// CloseGPIO closes the GPIO and SPI. If the configuration is set to use real
// hardware this function will panic if the hardware cannot be closed.
func CloseGPIO() {
	if c.CONFIG.RealHW {
		rpio.SpiEnd(rpio.Spi0)
		if err := rpio.Close(); err != nil {
			panic(err)
		}
	}
}

// SPIExchangeMultiplex exchanges data via SPI.
func SPIExchangeMultiplex(index string, write []byte) []byte {
	spiMutex.Lock()
	defer spiMutex.Unlock()

	cfg, found := spimultiplexcfg[index]
	if !found {
		panic("No SPI multiplexe device configuration with index " + index + " found in config file")
	} else {
		for _, pin := range cfg.low {
			pin.Low()
		}
		for _, pin := range cfg.high {
			pin.High()
		}
	}

	return spi.Exchange(write)
}

// Access a MCP3008 ADC via SPI.  If you have another ADC attached via
// the SPI multiplexer you only need to change this function here.
func ReadAdc(multiplex string, channel byte) int {
	write := []byte{1, (8 + channel) << 4, 0}
	read := SPIExchangeMultiplex(multiplex, write)
	return ((int(read[1]) & 3) << 8) + int(read[2])
}

// Access a WS2801 or APA102 LED stripe via SPI. If you have a different LED stripe
// attached via the SPI multiplexer you only need to change this
// function here.
func SetLedSegment(multiplex string, values []p.Led) {
	var display []byte

	if c.CONFIG.Hardware.LEDType == "ws2801" {
		display = make([]byte, 3*len(values))
		for idx := range values {
			display[3*idx] = byte(math.Min(float64(values[idx].Red)*float64(c.CONFIG.Hardware.Display.ColorCorrection[0]), 255))
			display[(3*idx)+1] = byte(math.Min(float64(values[idx].Green)*float64(c.CONFIG.Hardware.Display.ColorCorrection[1]), 255))
			display[(3*idx)+2] = byte(math.Min(float64(values[idx].Blue)*float64(c.CONFIG.Hardware.Display.ColorCorrection[2]), 255))
		}
	} else if c.CONFIG.Hardware.LEDType == "apa102" {
		// frame start: 4 zero bytes
		frameStart := []byte{0x00, 0x00, 0x00, 0x00}
		display = append(display, frameStart...)

		// Fixed general brightness
		brightness := byte(c.CONFIG.Hardware.Display.APA102_Brightness) | 0xE0

		// LED data
		for i := range values {
			red := byte(math.Min(float64(values[i].Red)*float64(c.CONFIG.Hardware.Display.ColorCorrection[0]), 255))
			green := byte(math.Min(float64(values[i].Green)*float64(c.CONFIG.Hardware.Display.ColorCorrection[1]), 255))
			blue := byte(math.Min(float64(values[i].Blue)*float64(c.CONFIG.Hardware.Display.ColorCorrection[2]), 255))

			// protocol: brightness byte
			display = append(display, brightness, blue, green, red)
		}

		// frame end: at least (len(values) / 2) + 1 bits of 0xFF
		// using number of bytes here
		frameEndLength := int(len(values)/16) + 1
		frameEnd := make([]byte, frameEndLength)
		for i := range frameEnd {
			frameEnd[i] = 0xFF
		}
		display = append(display, frameEnd...)
	} else {
		log.Println("No LED stripe type defined. Please check the configuration.")
		return
	}

	SPIExchangeMultiplex(multiplex, display)
}
