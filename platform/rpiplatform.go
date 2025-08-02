package platform

import (
	"fmt"
	"log"
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
}

func NewRaspberryPiPlatform(conf *config.Config) *RaspberryPiPlatform {
	readyChan := make(chan bool)
	close(readyChan) // For RPi, we are ready immediately.
	inst := &RaspberryPiPlatform{
		sensorStopChan: make(chan bool),
		readyChan:      readyChan,
	}
	inst.AbstractPlatform = newAbstractPlatform(conf, inst.DisplayLeds)
	return inst
}

func (s *RaspberryPiPlatform) Ready() <-chan bool {
	return s.readyChan
}

// SetSensorViewer attaches an optional TUI viewer for sensor data.
func (s *RaspberryPiPlatform) SetSensorViewer(v *SensorViewer) {
	s.sensorViewer = v
}

func (s *RaspberryPiPlatform) Start(ledWriter chan []producer.Led) error {
	log.Println("Initialise GPIO and Spi...")
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
		s.spimultiplexcfg[key] = gpiocfg{
			low:  low,
			high: high,
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

	s.initSensors(s.config.Hardware.Sensors)

	s.displayWg.Add(1)
	go s.displayDriver(ledWriter)

	s.sensorWg.Add(1)
	go s.sensorDriver()

	return nil
}

func (s *RaspberryPiPlatform) Stop() {
	// Signal goroutines to stop
	close(s.displayStopChan)
	close(s.sensorStopChan)

	// Wait for them to finish
	s.displayWg.Wait()
	s.sensorWg.Wait()

	// Now, safely close hardware
	rpio.SpiEnd(rpio.Spi0)
	if err := rpio.Close(); err != nil {
		log.Printf("Error closing rpio: %v", err)
	}
}

func (s *RaspberryPiPlatform) DisplayLeds(leds []producer.Led) {
	for _, segarray := range s.segments {
		for _, seg := range segarray {
			seg.setLeds(leds)
			if seg.visible {
				if err := s.ledDriver.write(seg, s.spiExchangeMultiplex); err != nil {
					log.Printf("Error writing to LED driver: %v", err)
				}
			}
		}
	}
}

func (s *RaspberryPiPlatform) spiExchangeMultiplex(index string, data []byte) []byte {
	s.spiMutex.Lock()
	defer s.spiMutex.Unlock()

	cfg, found := s.spimultiplexcfg[index]
	if !found {
		panic("No SPI multiplexe device configuration with index " + index + " found in config file")
	} else {
		for _, pin := range cfg.low {
			pin.Low()
		}
		for _, pin := range cfg.high {
			pin.High()
		}
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
}

func newWs2801Driver(displayConfig config.DisplayConfig) *ws2801Driver {
	return &ws2801Driver{displayConfig: displayConfig}
}

func (d *ws2801Driver) write(segment *segment, exchangeFunc func(string, []byte) []byte) error {
	var display []byte
	display = make([]byte, 3*len(segment.leds))
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
}

func newApa102Driver(displayConfig config.DisplayConfig) *apa102Driver {
	return &apa102Driver{displayConfig: displayConfig}
}

func (d *apa102Driver) write(segment *segment, exchangeFunc func(string, []byte) []byte) error {
	var display []byte

	// frame start: 4 zero bytes
	frameStart := []byte{0x00, 0x00, 0x00, 0x00}
	display = append(display, frameStart...)

	// Fixed general brightness
	brightness := byte(d.displayConfig.APA102_Brightness) | 0xE0

	// LED data
	for i := range segment.leds {
		red := byte(math.Min(float64(segment.leds[i].Red)*float64(d.displayConfig.ColorCorrection[0]), 255))
		green := byte(math.Min(float64(segment.leds[i].Green)*float64(d.displayConfig.ColorCorrection[1]), 255))
		blue := byte(math.Min(float64(segment.leds[i].Blue)*float64(d.displayConfig.ColorCorrection[2]), 255))

		// protocol: brightness byte
		display = append(display, brightness, blue, green, red)
	}

	// frame end: at least (len(values) / 2) + 1 bits of 0xFF
	// using number of bytes here
	frameEndLength := int(len(segment.leds)/16) + 1
	frameEnd := make([]byte, frameEndLength)
	for i := range frameEnd {
		frameEnd[i] = 0xFF
	}
	display = append(display, frameEnd...)

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
			log.Println("Ending SensorDriver go-routine (RPi)")
			return
		case <-ticker.C:
			for name, sensor := range s.sensors {
				value := sensor.smoothValue(s.readAdc(sensor.spimultiplex, sensor.adcChannel))
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