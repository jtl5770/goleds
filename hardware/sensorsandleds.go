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
	"_s0": NewSensor(0, 0, 0, 80),
	"_s1": NewSensor(69, 0, 7, 80),
	"_s2": NewSensor(70, 1, 0, 80),
	"_s3": NewSensor(124, 1, 5, 80),
}

var COLORCORR = []float64{1.0, 0.18, 0.02}

// end of tuneable part

var pin17, pin22, pin23, pin24 rpio.Pin
var REAL bool = false

func init() {
	args := os.Args
	if len(args) == 2 && args[1] == "REAL" {
		REAL = true
		if err := rpio.Open(); err != nil {
			panic(err)
		}
		if err := rpio.SpiBegin(rpio.Spi0); err != nil {
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
		if !REAL {
			simulateLed(0, led1)
			simulateLed(1, led2)
		} else {
			spiMutex.Lock()
			setLedSegment(0, led1)
			setLedSegment(1, led2)
			spiMutex.Unlock()
		}
	}
}

func SensorDriver(sensorReader chan string, sensors map[string]Sensor) {
	if !REAL {
		simulateSensors(sensorReader)
		return
	}
	sensorvalues := make(map[string]int)
	// sensormax := make(map[string]int)
	// var ordered []string
	// for idx, _ := range sensors {
	// 	ordered = append(ordered, idx)
	// }
	// ordered = sort.StringSlice(ordered)
	for {
		spiMutex.Lock()
		for name, sensor := range sensors {
			selectAdc(sensor.adc)
			sensorvalues[name] = sensor.smoothValue(readAdc(sensor.adcIndex))
		}
		spiMutex.Unlock()
		// var buf strings.Builder
		for name, value := range sensorvalues {
			// max := sensormax[name]
			// if value > max {
			// 	sensormax[name] = value
			// }
			if value > sensors[name].triggerLevel {
				sensorReader <- name
			}
		}
		// for _, idx := range ordered {
		// 	fmt.Fprintf(&buf, "%4d ", sensormax[idx])
		// }
		// fmt.Fprintf(&buf, "\r")
		// fmt.Print(buf.String())
		time.Sleep(5 * time.Millisecond)
	}
}

func readAdc(channel byte) int {
	buffer := []byte{1, (8 + channel) << 4, 0}
	rpio.SpiExchange(buffer)
	return ((int(buffer[1]) & 3) << 8) + int(buffer[2])
}

func setLedSegment(segmentID int, values []c.Led) {
	selectLed(segmentID)
	display := make([]byte, 3*len(values))
	for idx, led := range values {
		display[3*idx] = byte(math.Round(COLORCORR[0] * float64(led.Red)))
		display[(3*idx)+1] = byte(math.Round(COLORCORR[1] * float64(led.Green)))
		display[(3*idx)+2] = byte(math.Round(COLORCORR[2] * float64(led.Blue)))
	}
	rpio.SpiExchange(display)
	//time.Sleep(time.Millisecond)
}

func selectLed(index int) {
	if index == 0 {
		pin22.High()
		pin17.Low()
		pin23.High()
		pin24.High()
	} else if index == 1 {
		pin22.Low()
		pin17.High()
		pin23.High()
		pin24.High()
	} else {
		panic("No LED")
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

func simulateLed(segmentID int, values []c.Led) {
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
	if segmentID == 0 {
		fmt.Print("]       ")
	} else {
		fmt.Print("]\r")
	}

}

func intensity(s c.Led) byte {
	return byte(math.Round(float64(s.Red+s.Green+s.Blue) / 3.0))
}

func simulateSensors(sensorReader chan string) {
	sensorReader <- "_s0"
	time.Sleep(15 * time.Second)
	sensorReader <- "_s3"
	time.Sleep(20 * time.Second)
	sensorReader <- "_s1"
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
