package hardware

import (
	"sync"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
)

// constants and other values describing the hardware.

var Sensors map[string]Sensor

const (
	//SPI_SPEED = 976562
	SPI_SPEED = 1000000
)

// *** end of tuneable part ***

var pin17, pin22, pin23, pin24 rpio.Pin
var spiMutex sync.Mutex

func InitGpioAndSensors(firsttime bool) {
	if c.CONFIG.RealHW {
		if firsttime {
			if err := rpio.Open(); err != nil {
				panic(err)
			}
			if err := rpio.SpiBegin(rpio.Spi0); err != nil {
				panic(err)
			}
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
	}
	Sensors = map[string]Sensor{
		"_s0": NewSensor(0, 0, 0, c.CONFIG.Hardware.Sensors.TriggerLeft),
		"_s1": NewSensor(69, 0, 7, c.CONFIG.Hardware.Sensors.TriggerMidLeft),
		"_s2": NewSensor(70, 1, 0, c.CONFIG.Hardware.Sensors.TriggerMidRight),
		"_s3": NewSensor(124, 1, 5, c.CONFIG.Hardware.Sensors.TriggerRight)}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
