package hardware

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
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
	"_s3": NewSensor(124, 1, 5, 130),
}

// end of tuneable part

var pin17, pin22, pin23, pin24 rpio.Pin

func init() {
	name, _ := os.Hostname()
	if name == "pilab" {
		if err := rpio.Open(); err != nil {
			panic(err)
		}

		pin23 = rpio.Pin(23)
		pin23.Output()
		pin23.High()

		pin24 = rpio.Pin(24)
		pin24.Output()
		pin24.High()

		pin22 = rpio.Pin(22)
		pin22.Output()
		pin22.Low()

		pin17 = rpio.Pin(17)
		pin17.Output()
		pin17.Low()
	}
}

type Sensor struct {
	LedIndex     int
	adc          int
	adcIndex     byte
	triggerLevel int
	values       []int
}

var spiMutex sync.Mutex

func NewSensor(ledIndex int, adc int, adcIndex byte, triggerLevel int) Sensor {
	return Sensor{
		LedIndex:     ledIndex,
		adc:          adc,
		adcIndex:     adcIndex,
		triggerLevel: triggerLevel,
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
		if v.IsEmpty() {
			buf.WriteString(" ")
		} else if intensity(v) > 50 {
			buf.WriteString("*")
		} else {
			buf.WriteString("_")
		}
	}
	fmt.Print(buf.String())
	if segementID == 0 {
		fmt.Print("]       ")
	} else {
		fmt.Print("]\r")
	}
}

func intensity(s c.Led) byte {
	return byte(math.Round(float64(s.Red+s.Green+s.Blue) / 3.0))
}

// *****
// TODO: real hardware implementation
// *****
func SensorDriver(sensorReader chan string, sensors map[string]Sensor) {
	name, _ := os.Hostname()
	if name != "pilab" {
		simulateSensors(sensorReader)
		return
	}
	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		panic(err)
	}
	defer rpio.SpiEnd(rpio.Spi0)
	sensorvalues := make(map[string]int)
	for {
		spiMutex.Lock()
		for name, sensor := range sensors {
			adc := sensor.adc
			channel := sensor.adcIndex
			selectAdc(adc)
			value := readAdc(channel)
			sensorvalues[name] = sensor.smoothValue(value)
		}
		spiMutex.Unlock()
		for name, value := range sensorvalues {
			if value > sensors[name].triggerLevel {
				sensorReader <- name
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func selectAdc(index int) {
	if index == 0 {
		pin22.Low()
		pin17.Low()
		pin23.Low()
		pin24.High()
	} else if index == 1 {
		pin22.Low()
		pin17.Low()
		pin23.High()
		pin24.Low()
	} else {
		panic("No ADC")
	}
}

func readAdc(channel byte) int {
	buffer := []byte{1, (8 + channel) << 4, 0}
	rpio.SpiExchange(buffer)
	return ((int(buffer[1]) & 3) << 8) + int(buffer[2])
}

func simulateSensors(sensorReader chan string) {
	sensorReader <- "_s0"
	time.Sleep(15 * time.Second)
	sensorReader <- "_s3"
	time.Sleep(20 * time.Second)
	sensorReader <- "_s1"
	// time.Sleep(20 * time.Second)
	// sensorReader <- "_s3"
	// time.Sleep(13 * time.Second)
	// sensorReader <- "_s0"
	// time.Sleep(1 * time.Second)
	// sensorReader <- "_s3"
	// time.Sleep(10 * time.Second)
	// sensorReader <- "_s1"
	// time.Sleep(8 * time.Second)
	// sensorReader <- "_s0"
	// time.Sleep(20 * time.Second)
	// sensorReader <- "_s3"
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
