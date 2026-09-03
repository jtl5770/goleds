package platform

import (
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/util"
)

var (
	globalCalibMutex  sync.RWMutex
	globalCalibCurves = make(map[string][]CalibPoint)
)

type AbstractPlatform struct {
	config          *config.Config
	ledsEvent       *util.AtomicEvent[[]producer.Led]
	sensorEvents    chan *util.Trigger
	sensors         map[string]*sensor
	segments        map[string][]*segment
	displayFunc     func([]producer.Led)
	displayWg       sync.WaitGroup
	displayStopChan chan bool
	readyChan       chan bool
	isShuttingDown  atomic.Bool
	isCalibrating   atomic.Bool
	currentMaxRed   atomic.Int32
	ledBufferPool   *sync.Pool
}

func newAbstractPlatform(conf *config.Config, displayFunc func([]producer.Led)) *AbstractPlatform {
	return &AbstractPlatform{
		config:          conf,
		ledsEvent:       util.NewAtomicEvent[[]producer.Led](),
		sensorEvents:    make(chan *util.Trigger),
		sensors:         make(map[string]*sensor),
		displayFunc:     displayFunc,
		displayStopChan: make(chan bool),
		readyChan:       make(chan bool),
	}
}

func (s *AbstractPlatform) Ready() <-chan bool {
	return s.readyChan
}

func (s *AbstractPlatform) SetLeds(leds []producer.Led) {
	s.ledsEvent.Send(leds)
}

func (s *AbstractPlatform) GetSensorEvents() <-chan *util.Trigger {
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
	calibCurve   atomic.Pointer[[]CalibPoint]
	values       []int
	index        int
	sum          int
}

func (s *sensor) thresholdForRed(red int) int {
	ptr := s.calibCurve.Load()
	if ptr == nil || len(*ptr) == 0 {
		return math.MaxInt
	}
	curve := *ptr
	if red >= curve[0].Red {
		return curve[0].Threshold
	}
	if red <= curve[len(curve)-1].Red {
		return curve[len(curve)-1].Threshold
	}
	for i := 0; i < len(curve)-1; i++ {
		p1 := curve[i]
		p2 := curve[i+1]
		if red <= p1.Red && red >= p2.Red {
			span := p1.Red - p2.Red
			if span == 0 {
				return p1.Threshold
			}
			r := float64(red-p2.Red) / float64(span)
			return p2.Threshold + int(math.Round(r*float64(p1.Threshold-p2.Threshold)))
		}
	}
	return curve[len(curve)-1].Threshold
}

func (s *sensor) setCalibrationCurve(curve []CalibPoint) {
	newCurve := make([]CalibPoint, len(curve))
	copy(newCurve, curve)
	s.calibCurve.Store(&newCurve)
}

func (s *sensor) smoothedValue(value int) int {
	n := len(s.values)
	if n <= 1 {
		return value
	}
	s.sum += value - s.values[s.index]
	s.values[s.index] = value
	s.index++
	if s.index == n {
		s.index = 0
	}
	return (s.sum + n/2) / n
}

func (s *AbstractPlatform) initSensors(sensorConfig config.SensorsConfig) {
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
