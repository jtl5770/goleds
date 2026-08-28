package platform

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/producer"
	"lautenbacher.net/goleds/util"
)

type RaspberryPiPlatform struct {
	*AbstractPlatform
	ledDriver       ledDriver
	spiMutex        sync.Mutex
	spimultiplexcfg map[string]gpiocfg
	sensorViewer    *SensorViewer
	sensorWg        sync.WaitGroup
	sensorStopChan  chan bool
}

type gpiocfg struct {
	low  []rpio.Pin
	high []rpio.Pin
	cs   rpio.Pin
}

func NewRaspberryPiPlatform(conf *config.Config) *RaspberryPiPlatform {
	inst := &RaspberryPiPlatform{
		sensorStopChan: make(chan bool),
	}
	inst.AbstractPlatform = newAbstractPlatform(conf, inst.rpiDisplayFunc)
	return inst
}

// SetSensorViewer attaches an optional TUI viewer for sensor data.
func (s *RaspberryPiPlatform) SetSensorViewer(v *SensorViewer) {
	s.sensorViewer = v
}

func (s *RaspberryPiPlatform) Start(pool *sync.Pool) error {
	s.ledBufferPool = pool

	s.segments = parseDisplaySegments(s.config.Hardware.Display)

	slog.Info("Initialise GPIO and Spi...")
	if err := rpio.Open(); err != nil {
		return fmt.Errorf("failed to open rpio: %w", err)
	}
	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		return fmt.Errorf("failed to begin spi: %w", err)
	}

	rpio.SpiSpeed(s.config.Hardware.SPIFrequency)

	s.spimultiplexcfg = make(map[string]gpiocfg, len(s.config.Hardware.SpiMultiplexGPIO))

	for key, cfg := range s.config.Hardware.SpiMultiplexGPIO {
		low := make([]rpio.Pin, 0, len(cfg.Low))
		high := make([]rpio.Pin, 0, len(cfg.High))
		for _, pin := range cfg.Low {
			rpiopin := rpio.Pin(pin)
			rpiopin.Output()
			low = append(low, rpiopin)
		}
		for _, pin := range cfg.High {
			rpiopin := rpio.Pin(pin)
			rpiopin.Output()
			high = append(high, rpiopin)
		}
		var cs rpio.Pin
		if cfg.CS != 0 {
			cs = rpio.Pin(cfg.CS)
			cs.Output()
			cs.High()
		}

		s.spimultiplexcfg[key] = gpiocfg{
			low:  low,
			high: high,
			cs:   cs,
		}
	}

	switch strings.ToUpper(s.config.Hardware.LEDType) {
	case "APA102":
		s.ledDriver = newApa102Driver(s.config.Hardware.Display)
	case "WS2801":
		s.ledDriver = newWs2801Driver(s.config.Hardware.Display)
	default:
		return fmt.Errorf("unknown LED type: %s", s.config.Hardware.LEDType)
	}

	if s.sensorViewer != nil {
		go s.sensorViewer.Start()
	}

	s.initSensors(s.config.Hardware.Sensors)

	s.displayWg.Add(1)
	go s.displayDriver()

	s.sensorWg.Add(1)
	go s.sensorDriver()

	close(s.readyChan) // For RPi, we are ready immediately.
	return nil
}

func (s *RaspberryPiPlatform) Stop() {
	s.setInShutdown()

	// Signal goroutines to stop
	close(s.displayStopChan)
	close(s.sensorStopChan)

	// Wait for them to finish
	s.displayWg.Wait()
	s.sensorWg.Wait()

	// Now, safely close hardware
	rpio.SpiEnd(rpio.Spi0)
	if err := rpio.Close(); err != nil {
		slog.Error("Error closing rpio", "error", err)
	}

	// If there is a SensorViewer TUI, close it.
	if s.sensorViewer != nil {
		s.sensorViewer.Stop()
	}
}

func (s *RaspberryPiPlatform) rpiDisplayFunc(leds []producer.Led) {
	for _, segarray := range s.segments {
		for _, seg := range segarray {
			seg.setLeds(leds)
			if seg.visible {
				if err := s.ledDriver.write(seg, s.spiExchangeMultiplex); err != nil {
					slog.Error("Error writing to LED driver", "error", err)
				}
			}
		}
	}
}

func (s *RaspberryPiPlatform) Calibrate() error {
	if !s.isCalibrating.CompareAndSwap(false, true) {
		return fmt.Errorf("calibration already in progress")
	}
	defer s.isCalibrating.Store(false)

	calibCfg := s.config.Hardware.Sensors.Calibration
	sensorLedRGB := s.config.SensorLED.LedRGB

	steps := []struct {
		name  string
		ratio float64
	}{
		{name: "Red-1.00", ratio: 1.00},
		{name: "Red-0.66", ratio: 0.66},
		{name: "Red-0.33", ratio: 0.33},
		{name: "Dark", ratio: 0.0},
	}

	setAllLeds := func(r, g, b float64) {
		leds := make([]producer.Led, s.config.Hardware.Display.LedsTotal)
		for i := range leds {
			leds[i] = producer.Led{Red: r, Green: g, Blue: b}
		}
		s.rpiDisplayFunc(leds)
	}

	flashRed := func(times int) {
		for i := 0; i < times; i++ {
			setAllLeds(255, 0, 0)
			time.Sleep(300 * time.Millisecond)
			setAllLeds(0, 0, 0)
			time.Sleep(200 * time.Millisecond)
		}
	}

	flashBlue := func(times int) {
		for i := 0; i < times; i++ {
			setAllLeds(0, 0, 255)
			time.Sleep(300 * time.Millisecond)
			setAllLeds(0, 0, 0)
			time.Sleep(200 * time.Millisecond)
		}
	}

	slog.Info("Starting sensor calibration...")

	for {
		stepCurves := make(map[string][]CalibPoint)
		for name := range s.sensors {
			stepCurves[name] = make([]CalibPoint, 0, len(steps))
		}

		failed := false

		for _, step := range steps {
			r := sensorLedRGB[0] * step.ratio
			setAllLeds(r, 0, 0)
			time.Sleep(150 * time.Millisecond)

			stepStart := time.Now()
			samples := make(map[string][]int)
			for name := range s.sensors {
				samples[name] = make([]int, 0, int(calibCfg.StepDuration/s.config.Hardware.Sensors.LoopDelay)+10)
			}

			for time.Since(stepStart) < calibCfg.StepDuration {
				if s.isShuttingDown.Load() {
					setAllLeds(0, 0, 0)
					return fmt.Errorf("platform shutting down")
				}
				for name, sensor := range s.sensors {
					raw := s.readAdc(sensor.spimultiplex, sensor.adcChannel)
					smoothed := sensor.smoothedValue(raw)
					samples[name] = append(samples[name], smoothed)
				}
				time.Sleep(s.config.Hardware.Sensors.LoopDelay)
			}

			for name := range s.sensors {
				vals := samples[name]
				if len(vals) == 0 {
					failed = true
					break
				}
				sort.Ints(vals)
				minVal := vals[0]
				maxVal := vals[len(vals)-1]
				medianVal := vals[len(vals)/2]

				variance := maxVal - minVal
				if variance > calibCfg.OutlierThreshold {
					slog.Warn("Calibration step outlier detected", "sensor", name, "step", step.name, "variance", variance, "min", minVal, "max", maxVal)
					failed = true
					break
				}

				spread := maxVal - medianVal
				calcMargin := int(math.Round(calibCfg.DeviationFactor * float64(spread)))
				effectiveMargin := calibCfg.MinMargin
				if calcMargin > effectiveMargin {
					effectiveMargin = calcMargin
				}

				threshold := maxVal + effectiveMargin

				slog.Info("Calibration measurement",
					"step", step.name,
					"ratio", step.ratio,
					"sensor", name,
					"min", minVal,
					"median", medianVal,
					"max", maxVal,
					"spread", spread,
					"calcMargin", calcMargin,
					"effectiveMargin", effectiveMargin,
					"threshold", threshold,
				)

				normB := step.ratio * (sensorLedRGB[0] / 255.0)
				stepCurves[name] = append(stepCurves[name], CalibPoint{
					Brightness: normB,
					Threshold:  threshold,
				})
			}

			if failed {
				break
			}
		}

		if failed {
			flashRed(3)
			time.Sleep(calibCfg.RetryDelay)
			continue
		}

		globalCalibMutex.Lock()
		for name, curve := range stepCurves {
			s.sensors[name].setCalibrationCurve(curve)
			globalCalibCurves[name] = curve
			slog.Info("Calibrated sensor curve", "sensor", name, "curve", curve)
		}
		globalCalibMutex.Unlock()
		setAllLeds(0, 0, 0)
		flashBlue(2)
		slog.Info("Sensor calibration completed successfully.")
		break
	}

	return nil
}

func (s *RaspberryPiPlatform) spiExchangeMultiplex(index string, data []byte) []byte {
	s.spiMutex.Lock()
	defer s.spiMutex.Unlock()

	// The existence of the key is guaranteed by the config validation at startup.
	cfg := s.spimultiplexcfg[index]
	for _, pin := range cfg.low {
		pin.Low()
	}
	for _, pin := range cfg.high {
		pin.High()
	}
	if cfg.cs != rpio.Pin(0) {
		cfg.cs.Low()
		defer cfg.cs.High()
	}
	rpio.SpiExchange(data)
	return data
}

// ledDriver interface and implementations
type ledDriver interface {
	write(segment *segment, exchangeFunc func(string, []byte) []byte) error
}

type colorLUT struct {
	r [256]byte
	g [256]byte
	b [256]byte
}

func newColorLUT(correction []float64) colorLUT {
	var lut colorLUT
	corR, corG, corB := 1.0, 1.0, 1.0
	if len(correction) >= 3 {
		corR, corG, corB = correction[0], correction[1], correction[2]
	}
	for i := 0; i < 256; i++ {
		lut.r[i] = byte(math.Min(float64(i)*corR, 255))
		lut.g[i] = byte(math.Min(float64(i)*corG, 255))
		lut.b[i] = byte(math.Min(float64(i)*corB, 255))
	}
	return lut
}

type ws2801Driver struct {
	displayConfig config.DisplayConfig
	buffer        []byte
	lut           colorLUT
}

func newWs2801Driver(displayConfig config.DisplayConfig) *ws2801Driver {
	// Pre-allocate buffer to the maximum possible size.
	maxSize := 3 * displayConfig.LedsTotal
	return &ws2801Driver{
		displayConfig: displayConfig,
		buffer:        make([]byte, maxSize),
		lut:           newColorLUT(displayConfig.ColorCorrection),
	}
}

func (d *ws2801Driver) write(segment *segment, exchangeFunc func(string, []byte) []byte) error {
	requiredSize := 3 * len(segment.leds)
	display := d.buffer[:requiredSize]

	for idx, led := range segment.leds {
		display[3*idx] = d.lut.r[byte(led.Red)]
		display[(3*idx)+1] = d.lut.g[byte(led.Green)]
		display[(3*idx)+2] = d.lut.b[byte(led.Blue)]
	}
	exchangeFunc(segment.spiMultiplex, display)
	return nil
}

type apa102Driver struct {
	displayConfig config.DisplayConfig
	buffer        []byte
	lut           colorLUT
}

func newApa102Driver(displayConfig config.DisplayConfig) *apa102Driver {
	// Pre-allocate buffer to the maximum possible size.
	frameEndLength := (displayConfig.LedsTotal / 16) + 1
	maxSize := 4 + (4 * displayConfig.LedsTotal) + frameEndLength
	return &apa102Driver{
		displayConfig: displayConfig,
		buffer:        make([]byte, maxSize),
		lut:           newColorLUT(displayConfig.ColorCorrection),
	}
}

func (d *apa102Driver) write(segment *segment, exchangeFunc func(string, []byte) []byte) error {
	// Calculate required size for the current segment
	frameEndLength := (len(segment.leds) / 16) + 1
	requiredSize := 4 + (4 * len(segment.leds)) + frameEndLength
	display := d.buffer[:requiredSize]

	// Frame start: 4 zero bytes
	copy(display[0:4], []byte{0x00, 0x00, 0x00, 0x00})

	// Fixed general brightness
	brightness := byte(d.displayConfig.APA102_Brightness) | 0xE0

	// LED data
	offset := 4
	for _, led := range segment.leds {
		// protocol: brightness byte, blue, green, red
		display[offset] = brightness
		display[offset+1] = d.lut.b[byte(led.Blue)]
		display[offset+2] = d.lut.g[byte(led.Green)]
		display[offset+3] = d.lut.r[byte(led.Red)]
		offset += 4
	}

	// Frame end: fill the rest of the slice with 0xFF
	for i := offset; i < requiredSize; i++ {
		display[i] = 0xFF
	}

	exchangeFunc(segment.spiMultiplex, display)
	return nil
}

func (s *RaspberryPiPlatform) sensorDriver() {
	defer s.sensorWg.Done()
	ticker := time.NewTicker(s.config.Hardware.Sensors.LoopDelay)
	defer ticker.Stop()

	latestValues := make(map[string]int)

	for {
		select {
		case <-s.sensorStopChan:
			slog.Info("Ending SensorDriver go-routine (RPi)")
			return
		case <-ticker.C:
			if s.isCalibrating.Load() {
				continue
			}
			r := s.getCurrentMaxRed()
			for name, sensor := range s.sensors {
				value := sensor.smoothedValue(s.readAdc(sensor.spimultiplex, sensor.adcChannel))
				latestValues[name] = value
				if !s.isCalibrating.Load() {
					threshold := sensor.thresholdForBrightness(r)
					if value > threshold {
						s.sensorEvents <- util.NewTrigger(name, value, time.Now())
					}
				}
			}

			if s.sensorViewer != nil {
				s.sensorViewer.Update(latestValues)
			}
		}
	}
}

func (s *RaspberryPiPlatform) readAdc(multiplex string, channel byte) int {
	write := []byte{1, (8 + channel) << 4, 0}
	read := s.spiExchangeMultiplex(multiplex, write)
	return ((int(read[1]) & 3) << 8) + int(read[2])
}
