package hardware

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	c "lautenbacher.net/goleds/config"
)

type Sensor struct {
	LedIndex     int
	adc          int
	adcChannel   byte
	triggerValue int
	values       []int
}

type Trigger struct {
	ID        string
	Value     int
	Timestamp time.Time
}

func NewSensor(ledIndex int, adc int, adcChannel byte, triggerValue int) Sensor {
	smoothing := c.CONFIG.Hardware.Sensors.SmoothingSize
	return Sensor{
		LedIndex:     ledIndex,
		adc:          adc,
		adcChannel:   adcChannel,
		triggerValue: triggerValue,
		values:       make([]int, smoothing, smoothing+1),
	}
}

func (s *Sensor) smoothValue(val int) int {
	var ret int
	smoothing := c.CONFIG.Hardware.Sensors.SmoothingSize
	newValues := make([]int, smoothing, smoothing+1)
	for index, curr := range append(s.values, val)[1:] {
		newValues[index] = curr
		ret += curr
	}
	s.values = newValues
	return ret / smoothing
}

func SensorDriver(sensorReader chan Trigger, sensors map[string]Sensor, sig chan bool) {
	if !c.CONFIG.RealHW {
		simulateSensors(sensorReader, sig)
		return
	}
	statistics := make(chan os.Signal)
	signal.Notify(statistics, syscall.SIGUSR1)

	sensorvalues := make(map[string]int)
	sensormax := make(map[string]int)
	ticker := time.NewTicker(c.CONFIG.Hardware.Sensors.LoopDelayMillis * time.Millisecond)
	for {
		select {
		case <-statistics:
			printStatisticsAndReset(&sensormax)
		case <-sig:
			log.Println("Ending SensorDriver go-routine")
			ticker.Stop()
			return
		case <-ticker.C:
			spiMutex.Lock()
			for name, sensor := range sensors {
				selectAdc(sensor.adc)
				sensorvalues[name] = sensor.smoothValue(readAdc(sensor.adcChannel))
			}
			spiMutex.Unlock()
			for name, value := range sensorvalues {
				if value > sensormax[name] {
					sensormax[name] = value
				}
				if value > sensors[name].triggerValue {
					sensorReader <- Trigger{name, value, time.Now()}
				}
			}
		}
	}
}

func printStatisticsAndReset(max *map[string]int) {
	keys := make([]string, 0, len(*max))
	for k := range *max {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] < keys[j] {
			return true
		} else {
			return false
		}
	})
	var output string
	for _, name := range keys {
		output = output + fmt.Sprintf("[%3d] ", (*max)[name])
		delete(*max, name)
	}
	log.Print(output)
}

func readAdc(channel byte) int {
	buffer := []byte{1, (8 + channel) << 4, 0}
	rpio.SpiExchange(buffer)
	time.Sleep(50 * time.Microsecond)
	return ((int(buffer[1]) & 3) << 8) + int(buffer[2])
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
	time.Sleep(50 * time.Microsecond)
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
