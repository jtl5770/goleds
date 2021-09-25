package hardware

import (
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
	smoothing := c.CONFIG.Hardware.Sensors.SmoothingSize
	return Sensor{
		LedIndex:     ledIndex,
		adc:          adc,
		adcIndex:     adcIndex,
		triggerLevel: triggerLevel,
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
	var retval int
	ticker := time.NewTicker(c.CONFIG.Hardware.Sensors.LoopDelayMillis * time.Millisecond)
	for {
		select {
		case <-statistics:
			printstatistics(sensormax)
			for k := range sensormax {
				delete(sensormax, k)
			}
		case <-sig:
			log.Println("Ending SensorDriver go-routine")
			ticker.Stop()
			return
		case <-ticker.C:
			spiMutex.Lock()
			for name, sensor := range sensors {
				selectAdc(sensor.adc)
				retval = sensor.smoothValue(readAdc(sensor.adcIndex))
				sensorvalues[name] = retval
				if retval > sensormax[name] {
					sensormax[name] = retval
				}
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

func printstatistics(max map[string]int) {
	keys := make([]string, 0, len(max))
	for k := range max {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] < keys[j] {
			return true
		} else {
			return false
		}
	})
	log.Print("\n")
	for _, v := range keys {
		log.Printf("%4d  ", max[v])
	}
	log.Print("\n")
}

func readAdc(channel byte) int {
	buffer := []byte{1, (8 + channel) << 4, 0}
	rpio.SpiExchange(buffer)
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
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
