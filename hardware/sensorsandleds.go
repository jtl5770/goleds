package hardware

import (
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/producer"
)

// constants and other values describing the hardware.

var COLORCORR = []float64{1.0, 0.18, 0.02}
var Sensors map[string]Sensor

const (
	LEDS_TOTAL           = 125
	LEDS_SPLIT           = 70
	SMOOTHING_SIZE       = 3
	SENSOR_LOOP_DELAY_MS = 5
	SPI_SPEED            = 976562
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
		"_s0": NewSensor(0, 0, 0, c.CONFIG.Sensors.TriggerLeft),
		"_s1": NewSensor(69, 0, 7, c.CONFIG.Sensors.TriggerMidLeft),
		"_s2": NewSensor(70, 1, 0, c.CONFIG.Sensors.TriggerMidRight),
		"_s3": NewSensor(124, 1, 5, c.CONFIG.Sensors.TriggerRight)}
}

type Sensor struct {
	LedIndex     int
	adc          int
	adcIndex     byte
	triggerLevel int
	values       []int
}

type Trigger struct {
	ID        string
	Value     int
	Timestamp time.Time
}

func NewSensor(ledIndex int, adc int, adcIndex byte, triggerLevel int) Sensor {
	return Sensor{
		LedIndex:     ledIndex,
		adc:          adc,
		adcIndex:     adcIndex,
		triggerLevel: triggerLevel,
		values:       make([]int, SMOOTHING_SIZE, SMOOTHING_SIZE+1),
	}
}

func (s *Sensor) smoothValue(val int) int {
	var ret int
	newValues := make([]int, SMOOTHING_SIZE, SMOOTHING_SIZE+1)
	for index, curr := range append(s.values, val)[1:] {
		newValues[index] = curr
		ret += curr
	}
	s.values = newValues
	return ret / SMOOTHING_SIZE
}

func DisplayDriver(display chan ([]c.Led), sig chan bool) {
	for {
		select {
		case <-sig:
			log.Println("Ending DisplayDriver go-routine")
			return
		case sumLeds := <-display:
			led1 := sumLeds[:LEDS_SPLIT]
			led2 := sumLeds[LEDS_SPLIT:]
			if !c.CONFIG.RealHW {
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
}

func SensorDriver(sensorReader chan Trigger, sensors map[string]Sensor, sig chan bool) {
	if !c.CONFIG.RealHW {
		simulateSensors(sensorReader, sig)
		return
	}
	sensorvalues := make(map[string]int)
	ticker := time.NewTicker(SENSOR_LOOP_DELAY_MS * time.Millisecond)
	for {
		select {
		case <-sig:
			log.Println("Ending SensorDriver go-routine")
			ticker.Stop()
			return
		case <-ticker.C:
			spiMutex.Lock()
			for name, sensor := range sensors {
				selectAdc(sensor.adc)
				sensorvalues[name] = sensor.smoothValue(readAdc(sensor.adcIndex))
			}
			spiMutex.Unlock()
			for name, value := range sensorvalues {
				if value > sensors[name].triggerLevel {
					sensorReader <- Trigger{name, value, time.Now()}
				}
			}
		}
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

// This rest of the file is used to simulate Led runs without hardware

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

func simulateSensors(sensorReader chan Trigger, sig chan bool) {
	for {
		sensorReader <- Trigger{"_s0", 80, time.Now()}
		if !waitorbreak(15*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"_s3", 80, time.Now()}
		if !waitorbreak(20*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"_s1", 80, time.Now()}
		if !waitorbreak(15*time.Second, sig) {
			return
		}
		sensorReader <- Trigger{"_s2", 80, time.Now()}
		if !waitorbreak(15*time.Second, sig) {
			return
		}
	}
}

func waitorbreak(wait time.Duration, sig chan bool) bool {
	select {
	case <-time.After(wait):
		return true
	case <-sig:
		log.Println("Ending SensorDriver go-routine")
		return false
	}
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
