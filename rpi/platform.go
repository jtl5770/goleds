package rpi

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"github.com/stianeikeland/go-rpio/v4"
	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/platform"
	"lautenbacher.net/goleds/producer"
	"strings"
)

type RaspberryPiPlatform struct {
	config          *config.Config
	sensorEvents    chan *platform.Trigger
	stopChan        chan bool
	ledDriver       LEDDriver
	spiMutex        sync.Mutex
	spimultiplexcfg map[string]gpiocfg
	sensors         map[string]*sensor
	displayManager  *platform.DisplayManager
}

type gpiocfg struct {
	low  []rpio.Pin
	high []rpio.Pin
}

func NewPlatform(conf *config.Config) *RaspberryPiPlatform {
	return &RaspberryPiPlatform{
		config:       conf,
		sensorEvents: make(chan *platform.Trigger),
		stopChan:     make(chan bool),
	}
}

func (p *RaspberryPiPlatform) Start() error {
	log.Println("Initialise GPIO and Spi...")
	if err := rpio.Open(); err != nil {
		return fmt.Errorf("failed to open rpio: %w", err)
	}
	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		return fmt.Errorf("failed to begin spi: %w", err)
	}

	rpio.SpiSpeed(p.config.Hardware.SPIFrequency)

	p.spimultiplexcfg = make(map[string]gpiocfg, len(p.config.Hardware.SpiMultiplexGPIO))

	for key, cfg := range p.config.Hardware.SpiMultiplexGPIO {
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
		p.spimultiplexcfg[key] = gpiocfg{
			low:  low,
			high: high,
		}
	}

	p.displayManager = platform.NewDisplayManager(p.config.Hardware.Display)

	switch strings.ToUpper(p.config.Hardware.LEDType) {
	case "APA102":
		p.ledDriver = newAPA102Driver(p.config.Hardware.Display)
	case "WS2801":
		p.ledDriver = newWS2801Driver(p.config.Hardware.Display)
	default:
		return fmt.Errorf("unknown LED type: %s", p.config.Hardware.LEDType)
	}

	p.initSensors(p.config.Hardware.Sensors)
	

	return nil
}

func (p *RaspberryPiPlatform) Stop() {
	rpio.SpiEnd(rpio.Spi0)
	if err := rpio.Close(); err != nil {
		log.Printf("Error closing rpio: %v", err)
	}
	close(p.stopChan)
}

func (p *RaspberryPiPlatform) DisplayLeds(leds []producer.Led) {
	for _, segarray := range p.displayManager.Segments {
		for _, seg := range segarray {
			seg.SetLeds(leds)
			if seg.Visible {
				if err := p.ledDriver.Write(seg, p.spiExchangeMultiplex); err != nil {
					log.Printf("Error writing to LED driver: %v", err)
				}
			}
		}
	}
}

func (p *RaspberryPiPlatform) GetSensorEvents() <-chan *platform.Trigger {
	return p.sensorEvents
}

func (p *RaspberryPiPlatform) GetSensorLedIndices() map[string]int {
	indices := make(map[string]int)
	for uid, sensor := range p.sensors {
		indices[uid] = sensor.LedIndex
	}
	return indices
}

func (p *RaspberryPiPlatform) LedsTotal() int {
	return p.config.Hardware.Display.LedsTotal
}

func (p *RaspberryPiPlatform) ForceUpdateDelay() time.Duration {
	return p.config.Hardware.Display.ForceUpdateDelay
}

func (p *RaspberryPiPlatform) DisplayDriver(display chan []producer.Led, stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-stopSignal:
			log.Println("Ending DisplayDriver go-routine (RPi)")
			return
		case sumLeds := <-display:
			p.DisplayLeds(sumLeds)
		}
	}
}

func (p *RaspberryPiPlatform) spiExchangeMultiplex(index string, data []byte) []byte {
	p.spiMutex.Lock()
	defer p.spiMutex.Unlock()

	cfg, found := p.spimultiplexcfg[index]
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

// LEDDriver interface and implementations
type LEDDriver interface {
	Write(segment *platform.Segment, exchangeFunc func(string, []byte) []byte) error
}

type WS2801Driver struct {
	displayConfig config.DisplayConfig
}

func newWS2801Driver(displayConfig config.DisplayConfig) *WS2801Driver {
	return &WS2801Driver{displayConfig: displayConfig}
}

func (d *WS2801Driver) Write(segment *platform.Segment, exchangeFunc func(string, []byte) []byte) error {
	var display []byte
	display = make([]byte, 3*len(segment.Leds))
	for idx := range segment.Leds {
		display[3*idx] = byte(math.Min(float64(segment.Leds[idx].Red)*float64(d.displayConfig.ColorCorrection[0]), 255))
		display[(3*idx)+1] = byte(math.Min(float64(segment.Leds[idx].Green)*float64(d.displayConfig.ColorCorrection[1]), 255))
		display[(3*idx)+2] = byte(math.Min(float64(segment.Leds[idx].Blue)*float64(d.displayConfig.ColorCorrection[2]), 255))
	}
	exchangeFunc(segment.SpiMultiplex, display)
	return nil
}

type APA102Driver struct {
	displayConfig config.DisplayConfig
}

func newAPA102Driver(displayConfig config.DisplayConfig) *APA102Driver {
	return &APA102Driver{displayConfig: displayConfig}
}

func (d *APA102Driver) Write(segment *platform.Segment, exchangeFunc func(string, []byte) []byte) error {
	var display []byte

	// frame start: 4 zero bytes
	frameStart := []byte{0x00, 0x00, 0x00, 0x00}
	display = append(display, frameStart...)

	// Fixed general brightness
	brightness := byte(d.displayConfig.APA102_Brightness) | 0xE0

	// LED data
	for i := range segment.Leds {
		red := byte(math.Min(float64(segment.Leds[i].Red)*float64(d.displayConfig.ColorCorrection[0]), 255))
		green := byte(math.Min(float64(segment.Leds[i].Green)*float64(d.displayConfig.ColorCorrection[1]), 255))
		blue := byte(math.Min(float64(segment.Leds[i].Blue)*float64(d.displayConfig.ColorCorrection[2]), 255))

		// protocol: brightness byte
		display = append(display, brightness, blue, green, red)
	}

	// frame end: at least (len(values) / 2) + 1 bits of 0xFF
	// using number of bytes here
	frameEndLength := int(len(segment.Leds)/16) + 1
	frameEnd := make([]byte, frameEndLength)
	for i := range frameEnd {
		frameEnd[i] = 0xFF
	}
	display = append(display, frameEnd...)

	exchangeFunc(segment.SpiMultiplex, display)
	return nil
}

// sensor and related functions
type sensor struct {
	uid          string
	LedIndex     int
	spimultiplex string
	adcChannel   byte
	triggerValue int
	values       []int
	smoothing    int
}

func (p *RaspberryPiPlatform) initSensors(sensorConfig config.SensorsConfig) {
	p.sensors = make(map[string]*sensor, len(sensorConfig.SensorCfg))
	for uid, cfg := range sensorConfig.SensorCfg {
		p.sensors[uid] = newSensor(uid, cfg.LedIndex, cfg.SpiMultiplex, cfg.AdcChannel, cfg.TriggerValue, sensorConfig.SmoothingSize)
	}
}

func newSensor(uid string, ledIndex int, spimultiplex string, adcChannel byte, triggerValue int, smoothing int) *sensor {
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

func (p *RaspberryPiPlatform) SensorDriver(stopSignal chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	sensorvalues := make(map[string]*deque.Deque[int])
	for name := range p.sensors {
		sensorvalues[name] = &deque.Deque[int]{}
	}
	ticker := time.NewTicker(p.config.Hardware.Sensors.LoopDelay)
	for {
		select {
		case <-stopSignal:
			log.Println("Ending SensorDriver go-routine (RPi)")
			ticker.Stop()
			return
		case <-ticker.C:
			for name, sensor := range p.sensors {
				value := sensor.smoothValue(p.readAdc(sensor.spimultiplex, sensor.adcChannel))
				sensorvalues[name].PushBack(value)
				if sensorvalues[name].Len() > 500 {
					sensorvalues[name].PopFront()
				}
			}
			for name, values := range sensorvalues {
				val := values.Back()
				if val > p.sensors[name].triggerValue {
					p.sensorEvents <- platform.NewTrigger(name, val, time.Now())
				}
			}
		}
	}
}

func (p *RaspberryPiPlatform) readAdc(multiplex string, channel byte) int {
	write := []byte{1, (8 + channel) << 4, 0}
	read := p.spiExchangeMultiplex(multiplex, write)
	return ((int(read[1]) & 3) << 8) + int(read[2])
}
