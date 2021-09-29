package hardware

import (
	"log"
	"sync"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
)

const (
	//SPI_SPEED = 976562
	SPI_SPEED = 10000000
)

var Sensors map[string]Sensor
var pin17, pin22, pin23, pin24 rpio.Pin
var spiMutex sync.Mutex

func InitGpio() {
	if c.CONFIG.RealHW {
		log.Println("Initialise GPI and Spi...")
		if err := rpio.Open(); err != nil {
			panic(err)
		}
		if err := rpio.SpiBegin(rpio.Spi0); err != nil {
			panic(err)
		}

		rpio.SpiSpeed(SPI_SPEED)
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

// Local Variables:
// compile-command: "cd .. && go build"
// End:
