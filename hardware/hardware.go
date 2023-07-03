package hardware

import (
	"log"
	"sync"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
)

var (
	pin17, pin22, pin23, pin24 rpio.Pin
	spiMutex                   sync.Mutex
)

func InitGPIO() {
	if c.CONFIG.RealHW {
		log.Println("Initialise GPI and Spi...")
		if err := rpio.Open(); err != nil {
			panic(err)
		}
		if err := rpio.SpiBegin(rpio.Spi0); err != nil {
			panic(err)
		}

		rpio.SpiSpeed(c.CONFIG.Hardware.SPIFrequency)
		pin17 = rpio.Pin(17)
		pin17.Output()
		pin17.Low()

		pin22 = rpio.Pin(22)
		pin22.Output()
		pin22.Low()

		pin23 = rpio.Pin(23)
		pin23.Output()
		pin23.High()

		pin24 = rpio.Pin(24)
		pin24.Output()
		pin24.High()
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

func SPIExchange(write []byte) []byte {
	// time.Sleep(c.CONFIG.Hardware.SPIDelay)
	rpio.SpiExchange(write)
	// time.Sleep(c.CONFIG.Hardware.Display.SPIDelay)
	return write
}

func selectLed(index int) {
	if index == 0 {
		pin17.Low()
		pin22.High()
		pin23.High()
		pin24.High()
	} else if index == 1 {
		pin17.High()
		pin22.Low()
		pin23.High()
		pin24.High()
	} else {
		panic("No LED")
	}
}

func selectAdc(index int) {
	if index == 0 {
		pin17.Low()
		pin22.Low()
		pin23.Low()
		pin24.High()
	} else if index == 1 {
		pin17.Low()
		pin22.Low()
		pin23.High()
		pin24.Low()
	} else {
		panic("No ADC")
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
