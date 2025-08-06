package platform

import (
	"log/slog"
	"math"
	"sync"
	"time"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
	u "lautenbacher.net/goleds/util"
)

type AbstractPlatform struct {
	config          *c.Config
	sensorEvents    chan *u.Trigger
	sensors         map[string]*sensor
	segments        map[string][]*segment
	displayFunc     func([]p.Led)
	displayWg       sync.WaitGroup
	displayStopChan chan bool
	readyChan       chan bool
	shutdownMutex   sync.RWMutex
	isShuttingDown  bool
	ledBufferPool   *sync.Pool
}

func newAbstractPlatform(conf *c.Config, displayFunc func([]p.Led)) *AbstractPlatform {
	return &AbstractPlatform{
		config:          conf,
		sensorEvents:    make(chan *u.Trigger),
		sensors:         make(map[string]*sensor),
		segments:        parseDisplaySegments(conf.Hardware.Display),
		displayFunc:     displayFunc,
		displayStopChan: make(chan bool),
	}
}

func (s *AbstractPlatform) GetSensorEvents() <-chan *u.Trigger {
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

func (s *AbstractPlatform) setInShutdown() {
	s.shutdownMutex.Lock()
	s.isShuttingDown = true
	s.shutdownMutex.Unlock()
}

func (s *AbstractPlatform) displayDriver(display chan []p.Led) {
	defer s.displayWg.Done()
	for {
		select {
		case <-s.displayStopChan:
			slog.Info("Ending DisplayDriver go-routine...")
			return
		case sumLeds := <-display:
			s.shutdownMutex.RLock()
			if !s.isShuttingDown {
				s.displayFunc(sumLeds)
			}
			s.shutdownMutex.RUnlock()
			// Return the buffer to the pool for reuse.
			s.ledBufferPool.Put(sumLeds)
		}
	}
}

// sensor struct and related functions
type sensor struct {
	uid          string
	LedIndex     int
	spimultiplex string
	adcChannel   byte
	triggerValue int
	values       []int
	index        int
	sum          int
	capacity     int
}

func (s *sensor) smoothedValue(value int) int {
	oldValue := s.values[s.index]
	s.sum = s.sum - oldValue + value
	s.values[s.index] = value
	s.index = (s.index + 1) % s.capacity
	return int(math.Round(float64(s.sum) / float64(s.capacity)))
}

func (s *AbstractPlatform) initSensors(sensorConfig c.SensorsConfig) {
	s.sensors = make(map[string]*sensor, len(sensorConfig.SensorCfg))
	for uid, cfg := range sensorConfig.SensorCfg {
		s.sensors[uid] = newSensor(uid, cfg.LedIndex, cfg.SpiMultiplex, cfg.AdcChannel, cfg.TriggerValue, sensorConfig.SmoothingSize)
	}
}

func newSensor(uid string, ledIndex int, spimultiplex string, adcChannel byte, triggerValue int, smoothing int) *sensor {
	return &sensor{
		uid:          uid,
		LedIndex:     ledIndex,
		spimultiplex: spimultiplex,
		adcChannel:   adcChannel,
		triggerValue: triggerValue,
		values:       make([]int, smoothing),
		capacity:     smoothing,
	}
}
