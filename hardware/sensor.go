package hardware

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/gammazero/deque"
	"github.com/montanaflynn/stats"
	c "lautenbacher.net/goleds/config"
)

const STATS_SIZE = 500

var (
	Sensors      map[string]*Sensor
	SensorReader chan *Trigger
)

type Sensor struct {
	uid          string
	LedIndex     int
	spimultiplex int
	adcChannel   byte
	triggerValue int
	values       []int
}

func NewSensor(uid string, ledIndex int, spimultiplex int, adcChannel byte, triggerValue int) *Sensor {
	smoothing := c.CONFIG.Hardware.Sensors.SmoothingSize
	return &Sensor{
		uid:          uid,
		LedIndex:     ledIndex,
		spimultiplex: spimultiplex,
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

type Trigger struct {
	ID        string
	Value     int
	Timestamp time.Time
}

func NewTrigger(id string, value int, time time.Time) *Trigger {
	inst := Trigger{
		ID:        id,
		Value:     value,
		Timestamp: time,
	}
	return &inst
}

func InitSensors() {
	Sensors = make(map[string]*Sensor, len(c.CONFIG.Hardware.Sensors.SensorCfg))
	SensorReader = make(chan *Trigger)
	for uid, cfg := range c.CONFIG.Hardware.Sensors.SensorCfg {
		Sensors[uid] = NewSensor(uid, cfg.LedIndex, cfg.SpiMultiplex, cfg.AdcChannel, cfg.TriggerValue)
	}
}

func SensorDriver(stop chan bool) {
	if !c.CONFIG.RealHW && !c.CONFIG.SensorShow {
		// Sensor triggers will be simulated via key presses
		// we just wait for the signal on the stop channel and return
		select {
		case <-stop:
			log.Println("Ending SensorDriver go-routine")
			return
		}
	}
	rand.Seed(time.Now().UnixNano())
	sensorvalues := make(map[string]*deque.Deque[int])
	for name := range Sensors {
		sensorvalues[name] = deque.New[int]()
	}
	ticker := time.NewTicker(c.CONFIG.Hardware.Sensors.LoopDelay)
	for {
		select {
		case <-stop:
			log.Println("Ending SensorDriver go-routine")
			ticker.Stop()
			return
		case <-ticker.C:
			// spiMutex.Lock()
			for name, sensor := range Sensors {
				var value int
				if c.CONFIG.SensorShow && !c.CONFIG.RealHW {
					value = 30 + rand.Intn(250)
				} else {
					value = sensor.smoothValue(readAdc(sensor.spimultiplex, sensor.adcChannel))
				}
				sensorvalues[name].PushBack(value)
				if sensorvalues[name].Len() > STATS_SIZE {
					sensorvalues[name].PopFront()
				}
			}
			// spiMutex.Unlock()
			var buft strings.Builder
			var bufb strings.Builder
			buft.WriteString(" [min|mean|max]  ")
			bufb.WriteString(" (StdDev)        ")
			for name, value := range sensorvalues {
				val := value.Back()
				if c.CONFIG.SensorShow {
					data := make([]int, value.Len())
					for i := 0; i < value.Len(); i++ {
						data[i] = value.At(i)
					}
					stat := stats.LoadRawData(data)
					mean, _ := stat.Mean()
					stdev, _ := stat.StandardDeviation()
					max, _ := stat.Max()
					min, _ := stat.Min()
					buft.WriteString(fmt.Sprintf(" [%3.0f|%3.0f|%3.0f] ", min, mean, max)) // 15 chars
					bufb.WriteString(fmt.Sprintf(" (%5.1f)       ", stdev))                // 15 chars
					content.SetText(buft.String() + "\n" + bufb.String())
				} else {
					if val > Sensors[name].triggerValue {
						SensorReader <- NewTrigger(name, val, time.Now())
					}
				}
			}
		}
	}
}

func readAdc(multiplex int, channel byte) int {
	write := []byte{1, (8 + channel) << 4, 0}
	read := SPIExchangeMultiplex(multiplex, write)
	return ((int(read[1]) & 3) << 8) + int(read[2])
}
