package hardware

import (
	"fmt"
	"strings"
	"time"
)

const LEDS_TOTAL = 125
const _LEDS_SPLIT = 70

type Sensor struct {
	LedIndex int
	Adc      int
	AdcIndex int
}

var Sensors = []Sensor{{0, 0, 0}, {69, 0, 7}, {70, 1, 0}, {124, 1, 5}}

func NewSensor(ledIndex int, adc int, adcIndex int) Sensor {
	return Sensor{LedIndex: ledIndex, Adc: adc, AdcIndex: adcIndex}
}

// ****
// TODO: real hardware implementation
// ****
func DisplayDriver(display chan ([]byte)) {
	for {
		var tmp strings.Builder
		tmp.Grow(LEDS_TOTAL + 1)

		sumLeds := <-display
		led1 := sumLeds[0:_LEDS_SPLIT]
		led2 := sumLeds[_LEDS_SPLIT:]

		for _, v := range led1 {
			if v == 0 {
				tmp.WriteString(" ")
			} else {
				tmp.WriteString("*")
			}
		}
		tmp.WriteString("[            ]")
		for _, v := range led2 {
			if v == 0 {
				tmp.WriteString(" ")
			} else {
				tmp.WriteString("*")
			}
		}
		fmt.Print(tmp.String(), "\r")
	}
}

// ****
// TODO: real hardware implementation
// ****
func SensorDriver(sensorReader chan int, sensors []Sensor) {
	sensorReader <- 0
	time.Sleep(1 * time.Second)
	sensorReader <- 3
	time.Sleep(9 * time.Second)
	sensorReader <- 1
	time.Sleep(7 * time.Second)
	sensorReader <- 1
	time.Sleep(2 * time.Second)
	sensorReader <- 0
}
