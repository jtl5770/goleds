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

var (
	globalCalibMutex  sync.RWMutex
	globalCalibCurves = make(map[string][]CalibPoint)
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
	isShuttingDown  atomic.Bool
	isCalibrating   atomic.Bool
	currentMaxRed   atomic.Int32
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

func (s *AbstractPlatform) getCurrentMaxRed() int {
	return int(s.currentMaxRed.Load())
}

func (s *AbstractPlatform) setInShutdown() {
	s.isShuttingDown.Store(true)
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
			if !s.isShuttingDown.Load() && !s.isCalibrating.Load() {
				var maxR float64
				for _, led := range sumLeds {
					if led.Red > maxR {
						maxR = led.Red
					}
				}
				s.currentMaxRed.Store(int32(math.Round(maxR)))
				s.displayFunc(sumLeds)
			}
			// Return the buffer to the pool for reuse.
			s.ledBufferPool.Put(sumLeds)
		}
	}
}

type CalibPoint struct {
	Red       int
	Threshold int
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
}

func (s *sensor) hasCalibration() bool {
	s.calibMutex.RLock()
	defer s.calibMutex.RUnlock()
	return len(s.calibCurve) > 0
}

func (s *sensor) thresholdForRed(red int) int {
	s.calibMutex.RLock()
	defer s.calibMutex.RUnlock()
	if len(s.calibCurve) == 0 {
		return math.MaxInt
	}
	if red >= s.calibCurve[0].Red {
		return s.calibCurve[0].Threshold
	}
	if red <= s.calibCurve[len(s.calibCurve)-1].Red {
		return s.calibCurve[len(s.calibCurve)-1].Threshold
	}
	for i := 0; i < len(s.calibCurve)-1; i++ {
		p1 := s.calibCurve[i]
		p2 := s.calibCurve[i+1]
		if red <= p1.Red && red >= p2.Red {
			span := p1.Red - p2.Red
			if span == 0 {
				return p1.Threshold
			}
			r := float64(red-p2.Red) / float64(span)
			return p2.Threshold + int(math.Round(r*float64(p1.Threshold-p2.Threshold)))
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
	n := len(s.values)
	if n <= 1 {
		return value
	}
	s.sum += value - s.values[s.index]
	s.values[s.index] = value
	s.index = (s.index + 1) % n
	return (s.sum + n/2) / n
}

func (s *AbstractPlatform) initSensors(sensorConfig c.SensorsConfig) {
	s.sensors = make(map[string]*sensor, len(sensorConfig.SensorCfg))
	globalCalibMutex.RLock()
	defer globalCalibMutex.RUnlock()
	for uid, cfg := range sensorConfig.SensorCfg {
		sn := newSensor(uid, cfg.LedIndex, cfg.SpiMultiplex, cfg.AdcChannel, sensorConfig.SmoothingSize)
		if curve, ok := globalCalibCurves[uid]; ok {
			sn.setCalibrationCurve(curve)
		}
		s.sensors[uid] = sn
	}
}

func newSensor(uid string, ledIndex int, spimultiplex string, adcChannel byte, smoothing int) *sensor {
	return &sensor{
		uid:          uid,
		LedIndex:     ledIndex,
		spimultiplex: spimultiplex,
		adcChannel:   adcChannel,
		values:       make([]int, smoothing),
	}
}
