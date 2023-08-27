package hardware

import (
	"log"
	"sync"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
)

var (
	spiMutex        sync.Mutex
	spimultiplexcfg map[int]gpiocfg
)

type gpiocfg struct {
	low  []rpio.Pin
	high []rpio.Pin
}

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

		spimultiplexcfg = make(map[int]gpiocfg, len(c.CONFIG.Hardware.SpiMultiplexGPIO))

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

func CloseGPIO() {
	if c.CONFIG.RealHW {
		rpio.SpiEnd(rpio.Spi0)
		if err := rpio.Close(); err != nil {
			panic(err)
		}
	}
}

func SPIExchangeMultiplex(index int, write []byte) []byte {
	spiMutex.Lock()
	defer spiMutex.Unlock()

	cfg, found := spimultiplexcfg[index]
	if !found {
		panic("No SPI multiplexd device with index " + string(rune(index)) + " found. Valid values are 0,1,2,3")
	} else {
		for _, pin := range cfg.low {
			pin.Low()
		}
		for _, pin := range cfg.high {
			pin.High()
		}
	}

	rpio.SpiExchange(write)
	return write
}
