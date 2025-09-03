package platform

import (
	"fmt"
	"log/slog"
	"math"
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
	readyChan       chan bool
}

type gpiocfg struct {
	low  []rpio.Pin
	high []rpio.Pin
	cs   rpio.Pin
}

func NewRaspberryPiPlatform(conf *config.Config) *RaspberryPiPlatform {
	readyChan := make(chan bool)
	inst := &RaspberryPiPlatform{
		sensorStopChan: make(chan bool),
		readyChan:      readyChan,
	}
	inst.AbstractPlatform = newAbstractPlatform(conf, inst.rpiDisplayFunc)
	return inst
}

func (s *RaspberryPiPlatform) Ready() <-chan bool {
	return s.readyChan
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

type ws2801Driver struct {
	displayConfig config.DisplayConfig
	buffer        []byte
}

func newWs2801Driver(displayConfig config.DisplayConfig) *ws2801Driver {
	// Pre-allocate buffer to the maximum possible size.
	maxSize := 3 * displayConfig.LedsTotal
	return &ws2801Driver{
		displayConfig: displayConfig,
		buffer:        make([]byte, maxSize),
	}
}

func (d *ws2801Driver) write(segment *segment, exchangeFunc func(string, []byte) []byte) error {
	requiredSize := 3 * len(segment.leds)
	display := d.buffer[:requiredSize]

	for idx := range segment.leds {
		display[3*idx] = byte(math.Min(float64(segment.leds[idx].Red)*float64(d.displayConfig.ColorCorrection[0]), 255))
		display[(3*idx)+1] = byte(math.Min(float64(segment.leds[idx].Green)*float64(d.displayConfig.ColorCorrection[1]), 255))
		display[(3*idx)+2] = byte(math.Min(float64(segment.leds[idx].Blue)*float64(d.displayConfig.ColorCorrection[2]), 255))
	}
	exchangeFunc(segment.spiMultiplex, display)
	return nil
}

type apa102Driver struct {
	displayConfig config.DisplayConfig
	buffer        []byte
}

func newApa102Driver(displayConfig config.DisplayConfig) *apa102Driver {
	// Pre-allocate buffer to the maximum possible size.
	frameEndLength := (displayConfig.LedsTotal / 16) + 1
	maxSize := 4 + (4 * displayConfig.LedsTotal) + frameEndLength
	return &apa102Driver{
		displayConfig: displayConfig,
		buffer:        make([]byte, maxSize),
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
	for i := range segment.leds {
		red := byte(math.Min(float64(segment.leds[i].Red)*float64(d.displayConfig.ColorCorrection[0]), 255))
		green := byte(math.Min(float64(segment.leds[i].Green)*float64(d.displayConfig.ColorCorrection[1]), 255))
		blue := byte(math.Min(float64(segment.leds[i].Blue)*float64(d.displayConfig.ColorCorrection[2]), 255))

		// protocol: brightness byte, blue, green, red
		display[offset] = brightness
		display[offset+1] = blue
		display[offset+2] = green
		display[offset+3] = red
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
			for name, sensor := range s.sensors {
				value := sensor.smoothedValue(s.readAdc(sensor.spimultiplex, sensor.adcChannel))
				latestValues[name] = value
				if value > sensor.triggerValue {
					s.sensorEvents <- util.NewTrigger(name, value, time.Now())
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
