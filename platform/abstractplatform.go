package platform

import (
	"log"
	"sync"
	"time"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
)

type AbstractPlatform struct {
	config         *c.Config
	sensorEvents   chan *Trigger
	sensors        map[string]*sensor
	displayManager *DisplayManager
	displayFunc    func([]p.Led)
}

func NewAbstractPlatform(conf *c.Config, displayFunc func([]p.Led)) *AbstractPlatform {
	return &AbstractPlatform{
		config:         conf,
		sensorEvents:   make(chan *Trigger),
		sensors:        make(map[string]*sensor),
		displayManager: NewDisplayManager(conf.Hardware.Display),
		displayFunc:    displayFunc,
	}
}

func (s *AbstractPlatform) GetSensorEvents() <-chan *Trigger {
	return s.sensorEvents
}

func (s *AbstractPlatform) GetSensorLedIndices() map[string]int {
	indices := make(map[string]int)
	for uid, sensor := range s.sensors {
		indices[uid] = sensor.LedIndex
	}
	return indices
}

func (s *AbstractPlatform) GetLedsTotal() int {
	return s.config.Hardware.Display.LedsTotal
}

func (s *AbstractPlatform) GetForceUpdateDelay() time.Duration {
	return s.config.Hardware.Display.ForceUpdateDelay
}

func (s *AbstractPlatform) DisplayDriver(display chan []p.Led, stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-stopSignal:
			log.Println("Ending DisplayDriver go-routine...")
			return
		case sumLeds := <-display:
			s.displayFunc(sumLeds)
		}
	}
}

// sensor struct and related functions (now internal to TUIPlatform)
type sensor struct {
	uid          string
	LedIndex     int
	spimultiplex string
	adcChannel   byte
	triggerValue int
	values       []int
	smoothing    int
}

func (s *sensor) smoothValue(val int) int {
	var ret int
	newValues := make([]int, s.smoothing, s.smoothing+1)
	for index, curr := range append(s.values, val)[1:] {
		newValues[index] = curr
		ret += curr
	}
	s.values = newValues
	return ret / s.smoothing
}

func (s *AbstractPlatform) initSensors(sensorConfig c.SensorsConfig) {
	s.sensors = make(map[string]*sensor, len(sensorConfig.SensorCfg))
	for uid, cfg := range sensorConfig.SensorCfg {
		s.sensors[uid] = s.newSensor(uid, cfg.LedIndex, cfg.SpiMultiplex, cfg.AdcChannel, cfg.TriggerValue, sensorConfig.SmoothingSize)
	}
}

func (s *AbstractPlatform) newSensor(uid string, ledIndex int, spimultiplex string, adcChannel byte, triggerValue int, smoothing int) *sensor {
	return &sensor{
		uid:          uid,
		LedIndex:     ledIndex,
		spimultiplex: spimultiplex,
		adcChannel:   adcChannel,
		triggerValue: triggerValue,
		values:       make([]int, smoothing, smoothing+1),
		smoothing:    smoothing,
	}
}
