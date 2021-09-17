package hardware

import (
	"fmt"
	"strings"
	"sync"
	"time"

	c "lautenbacher.net/goleds/controller"
)

// constants and other values describing the hardware.
const (
	LEDS_TOTAL      = 125
	_LEDS_SPLIT     = 70
	_SMOOTHING_SIZE = 3
)

var Sensors = map[string]Sensor{
	"_s0": NewSensor(0, 0, 0, 120),
	"_s1": NewSensor(69, 0, 7, 130),
	"_s2": NewSensor(70, 1, 0, 140),
	"_s3": NewSensor(1240, 1, 5, 130),
}

// end of tuneable part

type Sensor struct {
	LedIndex     int
	adc          int
	adcIndex     int
	triggerLevel int
	values       []int
}

var spiMutex sync.Mutex

func NewSensor(ledIndex int, adc int, adcIndex int, trigger int) Sensor {
	return Sensor{
		LedIndex:     ledIndex,
		adc:          adc,
		adcIndex:     adcIndex,
		triggerLevel: trigger,
		values:       make([]int, _SMOOTHING_SIZE, _SMOOTHING_SIZE+1),
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

// *****
// TODO:  real hardware implementation
// *****
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

// *****
// TODO: real hardware implementation
// *****
func SensorDriver(sensorReader chan string, sensors map[string]Sensor) {
	sensorReader <- "s0"
	time.Sleep(11 * time.Second)
	sensorReader <- "s1"
	time.Sleep(11 * time.Second)
	sensorReader <- "s2"
	time.Sleep(11 * time.Second)
	sensorReader <- "s3"
	time.Sleep(11 * time.Second)
	sensorReader <- "s0"
	time.Sleep(1 * time.Second)
	sensorReader <- "s3"
	time.Sleep(10 * time.Second)
	sensorReader <- "s1"
	time.Sleep(7 * time.Second)
	sensorReader <- "s1"
	time.Sleep(2 * time.Second)
	sensorReader <- "s0"
	time.Sleep(15 * time.Second)
	sensorReader <- "s3"
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
