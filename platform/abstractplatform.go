package platform

import (
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	c "lautenbacher.net/goleds/config"
	p "lautenbacher.net/goleds/producer"
	u "lautenbacher.net/goleds/util"
)

type AbstractPlatform struct {
	config          *c.Config
	ledsEvent       *u.AtomicEvent[[]p.Led]
	sensorEvents    chan *u.Trigger
	sensors         map[string]*sensor
	segments        map[string][]*segment
	displayFunc     func([]p.Led)
	displayWg       sync.WaitGroup
	displayStopChan chan bool
	readyChan       chan bool
	shutdownMutex   sync.RWMutex
	isShuttingDown  bool
	isCalibrating   atomic.Bool
	brightnessMutex sync.RWMutex
	currentMaxR     float64
	currentMaxG     float64
	currentMaxB     float64
	ledBufferPool   *sync.Pool
}

func newAbstractPlatform(conf *c.Config, displayFunc func([]p.Led)) *AbstractPlatform {
	return &AbstractPlatform{
		config:          conf,
		ledsEvent:       u.NewAtomicEvent[[]p.Led](),
		sensorEvents:    make(chan *u.Trigger),
		sensors:         make(map[string]*sensor),
		displayFunc:     displayFunc,
		displayStopChan: make(chan bool),
		readyChan:       make(chan bool),
	}
}

func (s *AbstractPlatform) Ready() <-chan bool {
	return s.readyChan
}

func (s *AbstractPlatform) SetLeds(leds []p.Led) {
	s.ledsEvent.Send(leds)
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

func (s *AbstractPlatform) IsCalibrating() bool {
	return s.isCalibrating.Load()
}

func (s *AbstractPlatform) getCurrentMaxRed() float64 {
	s.brightnessMutex.RLock()
	defer s.brightnessMutex.RUnlock()
	return s.currentMaxR
}

func (s *AbstractPlatform) getCurrentMaxBrightness() float64 {
	s.brightnessMutex.RLock()
	defer s.brightnessMutex.RUnlock()
	return s.currentMaxB
}

func (s *AbstractPlatform) setInShutdown() {
	s.shutdownMutex.Lock()
	s.isShuttingDown = true
	s.shutdownMutex.Unlock()
}

func (s *AbstractPlatform) displayDriver() {
	defer s.displayWg.Done()
	for {
		select {
		case <-s.displayStopChan:
			slog.Info("Ending DisplayDriver go-routine...")
			return
		case <-s.ledsEvent.Channel():
			sumLeds := s.ledsEvent.Value()
			s.shutdownMutex.RLock()
			if !s.isShuttingDown && !s.isCalibrating.Load() {
				var maxR, maxG, maxB float64
				for _, led := range sumLeds {
					if led.Red > maxR {
						maxR = led.Red
					}
					if led.Green > maxG {
						maxG = led.Green
					}
					if led.Blue > maxB {
						maxB = led.Blue
					}
				}
				s.brightnessMutex.Lock()
				s.currentMaxR = maxR / 255.0
				s.currentMaxG = maxG / 255.0
				s.currentMaxB = maxB / 255.0
				s.brightnessMutex.Unlock()

				s.displayFunc(sumLeds)
			}
			s.shutdownMutex.RUnlock()
			// Return the buffer to the pool for reuse.
			s.ledBufferPool.Put(sumLeds)
		}
	}
}

type CalibPoint struct {
	Brightness float64
	Threshold  int
}

// sensor struct and related functions
type sensor struct {
	uid          string
	LedIndex     int
	spimultiplex string
	adcChannel   byte
	calibCurve   []CalibPoint
	calibMutex   sync.RWMutex
	values       []int
	index        int
	sum          int
	capacity     int
}

func (s *sensor) thresholdForBrightness(b float64) int {
	s.calibMutex.RLock()
	defer s.calibMutex.RUnlock()
	if len(s.calibCurve) == 0 {
		return 100
	}
	if b <= s.calibCurve[0].Brightness {
		return s.calibCurve[0].Threshold
	}
	if b >= s.calibCurve[len(s.calibCurve)-1].Brightness {
		return s.calibCurve[len(s.calibCurve)-1].Threshold
	}
	for i := 0; i < len(s.calibCurve)-1; i++ {
		p1 := s.calibCurve[i]
		p2 := s.calibCurve[i+1]
		if b >= p1.Brightness && b <= p2.Brightness {
			ratio := (b - p1.Brightness) / (p2.Brightness - p1.Brightness)
			return p1.Threshold + int(math.Round(ratio*float64(p2.Threshold-p1.Threshold)))
		}
	}
	return s.calibCurve[len(s.calibCurve)-1].Threshold
}

func (s *sensor) setCalibrationCurve(curve []CalibPoint) {
	s.calibMutex.Lock()
	defer s.calibMutex.Unlock()
	s.calibCurve = make([]CalibPoint, len(curve))
	copy(s.calibCurve, curve)
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
		s.sensors[uid] = newSensor(uid, cfg.LedIndex, cfg.SpiMultiplex, cfg.AdcChannel, sensorConfig.SmoothingSize)
	}
}

func newSensor(uid string, ledIndex int, spimultiplex string, adcChannel byte, smoothing int) *sensor {
	return &sensor{
		uid:          uid,
		LedIndex:     ledIndex,
		spimultiplex: spimultiplex,
		adcChannel:   adcChannel,
		values:       make([]int, smoothing),
		capacity:     smoothing,
	}
}
