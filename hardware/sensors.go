package hardware

import (
	"fmt"
	"time"
)

const LEDS_TOTAL = 125

var Sensors = []Sensor{{0, 0, 0}, {69, 0, 7}, {70, 1, 0}, {124, 1, 5}}

type Sensor struct {
	LedIndex int
	Adc      int
	AdcIndex int
}

func NewSensor(ledIndex int, adc int, adcIndex int) Sensor {
	return Sensor{LedIndex: ledIndex, Adc: adc, AdcIndex: adcIndex}
}

// ****
// TODO: real hardware implementation
// ****
func DisplayDriver(display chan ([]byte)) {
	for {
		sumLeds := <-display
		for _, v := range sumLeds {
			if v == 0 {
				fmt.Print(" ")
			} else {
				fmt.Print("☼")
			}
		}
		fmt.Print("\r")
	}
}

// ****
// TODO: real hardware implementation
// ****
func SensorDriver(sensorReader chan int, sensors []Sensor) {
	sensorReader <- 3
	sensorReader <- 0
	time.Sleep(9 * time.Second)
	sensorReader <- 1
	time.Sleep(7 * time.Second)
	sensorReader <- 1
	time.Sleep(2 * time.Second)
	sensorReader <- 0
}
