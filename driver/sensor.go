package driver

import (
	"log"
	"math/rand"
	"time"

	"github.com/gammazero/deque"
	c "lautenbacher.net/goleds/config"
	hw "lautenbacher.net/goleds/hardware"
)

const STATS_SIZE = 500

var (
	Sensors      map[string]*Sensor
	SensorReader chan *Trigger
)

type Sensor struct {
	uid          string
	LedIndex     int
	spimultiplex string
	adcChannel   byte
	triggerValue int
	values       []int
}

func NewSensor(uid string, ledIndex int, spimultiplex string, adcChannel byte, triggerValue int) *Sensor {
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
		// SimulationTUI is shown: Sensor triggers will be simulated
		// via key presses we just wait for the signal on the stop
		// channel and return
		select {
		case <-stop:
			log.Println("Ending SensorDriver go-routine")
			return
		}
	}
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
			for name, sensor := range Sensors {
				var value int
				if c.CONFIG.SensorShow && !c.CONFIG.RealHW {
					// Sensor measuring TUI is shown but sensor values will be simulated
					// this is only for testing the sensor measuring mode of the TUI
					value = 30 + rand.Intn(250)
				} else {
					value = sensor.smoothValue(hw.ReadAdc(sensor.spimultiplex, sensor.adcChannel))
				}
				sensorvalues[name].PushBack(value)
				if sensorvalues[name].Len() > STATS_SIZE {
					sensorvalues[name].PopFront()
				}
			}
			if c.CONFIG.SensorShow {
				sensorDisplay(sensorvalues)
			}
			for name, values := range sensorvalues {
				val := values.Back()
				if val > Sensors[name].triggerValue {
					SensorReader <- NewTrigger(name, val, time.Now())
				}
			}
		}
	}
}
