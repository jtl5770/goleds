package hardware

import (
	"fmt"
	"strings"
	"sync"
	"time"

	c "goleds/controller"
)

// constants and other values describing the hardware.
const (
	LEDS_TOTAL      = 125
	_LEDS_SPLIT     = 70
	_SMOOTHING_SIZE = 3
)

func init() {
	Sensors[0] = NewSensor(0, 0, 0, 120)
	Sensors[1] = NewSensor(69, 0, 7, 130)
	Sensors[2] = NewSensor(70, 1, 0, 140)
	Sensors[3] = NewSensor(124, 1, 5, 130)
}

// end of tuneable part

type Sensor struct {
	LedIndex int
	Adc      int
	AdcIndex int
	trigger  int
	values   []int
}

var spiMutex sync.Mutex
var Sensors []Sensor = make([]Sensor, 4)

func NewSensor(ledIndex int, adc int, adcIndex int, trigger int) Sensor {
	return Sensor{
		LedIndex: ledIndex,
		Adc:      adc,
		AdcIndex: adcIndex,
		trigger:  trigger,
		values:   make([]int, _SMOOTHING_SIZE, _SMOOTHING_SIZE+1),
	}
}

func (s *Sensor) smoothValue(val int) int {
	var ret int
	newValues := make([]int, _SMOOTHING_SIZE, _SMOOTHING_SIZE+1)
	for index, curr := range append(s.values, val)[1:] {
		newValues[index] = curr
		ret += curr
	}
	s.values = newValues
	return ret / _SMOOTHING_SIZE
}

func DisplayDriver(display chan ([]c.Led)) {
	for {
		sumLeds := <-display
		led1 := sumLeds[:_LEDS_SPLIT]
		led2 := sumLeds[_LEDS_SPLIT:]

		spiMutex.Lock()
		setLedSegment(0, led1)
		setLedSegment(1, led2)
		spiMutex.Unlock()
	}
}

// ****
// TODO: real hardware implementation
// ****
func setLedSegment(segementID int, values []c.Led) {
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
	time.Sleep(11 * time.Second)
	sensorReader <- 1
	time.Sleep(11 * time.Second)
	sensorReader <- 2
	time.Sleep(11 * time.Second)
	sensorReader <- 3
	time.Sleep(11 * time.Second)
	sensorReader <- 0
	time.Sleep(1 * time.Second)
	sensorReader <- 3
	time.Sleep(10 * time.Second)
	sensorReader <- 1
	time.Sleep(7 * time.Second)
	sensorReader <- 1
	time.Sleep(2 * time.Second)
	sensorReader <- 0
	time.Sleep(15 * time.Second)
	sensorReader <- 3
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
