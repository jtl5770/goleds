package hardware

import (
	"fmt"
	"strings"
	"sync"
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

var hwMutex sync.Mutex

func NewSensor(ledIndex int, adc int, adcIndex int) Sensor {
	return Sensor{LedIndex: ledIndex, Adc: adc, AdcIndex: adcIndex}
}

func DisplayDriver(display chan ([]byte)) {
	for {
		sumLeds := <-display
		led1 := sumLeds[0:_LEDS_SPLIT]
		led2 := sumLeds[_LEDS_SPLIT:]

		hwMutex.Lock()
		setLedSegment(0, led1)
		setLedSegment(1, led2)
		hwMutex.Unlock()
	}
}

// ****
// TODO: real hardware implementation
// ****
func setLedSegment(segementID int, values []byte) {
	var buf strings.Builder
	buf.Grow(len(values))

	fmt.Print("[")
	for _, v := range values {
		if v == 0 {
			buf.WriteString(" ")
		} else {
			buf.WriteString("*")
		}
	}
	fmt.Print(buf.String())
	if segementID == 0 {
		fmt.Print("]       ")
	} else {
		fmt.Print("]\r")
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

// Local Variables:
// compile-command: "cd .. && go build"
// End:
