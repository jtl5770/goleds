package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const CONFILE = "config.yml"

var CONFIG *Config

// SensorLEDConfig defines the configuration for the SensorLED producer.
type SensorLEDConfig struct {
	Enabled      bool          `yaml:"Enabled"`
	RunUpDelay   time.Duration `yaml:"RunUpDelay"`
	RunDownDelay time.Duration `yaml:"RunDownDelay"`
	HoldTime     time.Duration `yaml:"HoldTime"`
	LedRGB       []float64     `yaml:"LedRGB"`
}

// NightLEDConfig defines the configuration for the NightLED producer.
type NightLEDConfig struct {
	Enabled   bool        `yaml:"Enabled"`
	Latitude  float64     `yaml:"Latitude"`
	Longitude float64     `yaml:"Longitude"`
	LedRGB    [][]float64 `yaml:"LedRGB"`
}

// CylonLEDConfig defines the configuration for the CylonLED producer.
type CylonLEDConfig struct {
	Enabled  bool          `yaml:"Enabled"`
	Duration time.Duration `yaml:"Duration"`
	Delay    time.Duration `yaml:"Delay"`
	Step     float64       `yaml:"Step"`
	Width    int           `yaml:"Width"`
	LedRGB   []float64     `yaml:"LedRGB"`
}

// MultiBlobLEDConfig defines the configuration for the MultiBlobLED producer.
type MultiBlobLEDConfig struct {
	Enabled  bool               `yaml:"Enabled"`
	Duration time.Duration      `yaml:"Duration"`
	Delay    time.Duration      `yaml:"Delay"`
	BlobCfg  map[string]BlobCfg `yaml:"BlobCfg"`
}

// BlobCfg defines the configuration for a single blob in the MultiBlobLED producer.
type BlobCfg struct {
	DeltaX float64   `yaml:"DeltaX"`
	X      float64   `yaml:"X"`
	Width  float64   `yaml:"Width"`
	LedRGB []float64 `yaml:"LedRGB"`
}

// HardwareConfig defines the hardware configuration.
type HardwareConfig struct {
	LEDType          string        `yaml:"LEDType"`
	SPIFrequency     int           `yaml:"SPIFrequency"`
	Display          DisplayConfig `yaml:"Display"`
	Sensors          SensorsConfig `yaml:"Sensors"`
	SpiMultiplexGPIO map[string]struct {
		Low  []int `yaml:"Low"`
		High []int `yaml:"High"`
	} `yaml:"SpiMultiplexGPIO"`
}

// DisplayConfig defines the display configuration.
type DisplayConfig struct {
	ForceUpdateDelay  time.Duration                 `yaml:"ForceUpdateDelay"`
	LedsTotal         int                           `yaml:"LedsTotal"`
	ColorCorrection   []float64                     `yaml:"ColorCorrection"`
	APA102_Brightness byte                          `yaml:"APA102_Brightness"`
	LedSegments       map[string][]LedSegmentConfig `yaml:"LedSegments"`
}

// LedSegmentConfig defines the configuration for a single LED segment.
type LedSegmentConfig struct {
	FirstLed     int    `yaml:"FirstLed"`
	LastLed      int    `yaml:"LastLed"`
	SpiMultiplex string `yaml:"SpiMultiplex"`
	Reverse      bool   `yaml:"Reverse"`
}

// SensorCfg defines the configuration for a single sensor.
type SensorCfg struct {
	LedIndex     int    `yaml:"LedIndex"`
	SpiMultiplex string `yaml:"SpiMultiplex"`
	AdcChannel   byte   `yaml:"AdcChannel"`
	TriggerValue int    `yaml:"TriggerValue"`
}

// SensorsConfig defines the sensors configuration.
type SensorsConfig struct {
	SmoothingSize int                  `yaml:"SmoothingSize"`
	LoopDelay     time.Duration        `yaml:"LoopDelay"`
	SensorCfg     map[string]SensorCfg `yaml:"SensorCfg"`
}

type Config struct {
	RealHW       bool
	SensorShow   bool
	Configfile   string
	SensorLED    SensorLEDConfig    `yaml:"SensorLED"`
	NightLED     NightLEDConfig     `yaml:"NightLED"`
	CylonLED     CylonLEDConfig     `yaml:"CylonLED"`
	MultiBlobLED MultiBlobLEDConfig `yaml:"MultiBlobLED"`
	Hardware     HardwareConfig     `yaml:"Hardware"`
}

func ReadConfig(cfile string, realhw bool, sensorshow bool) *Config {
	log.Printf("Reading config file %s...", cfile)
	f, err := os.Open(cfile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	var conf Config
	err = decoder.Decode(&conf)
	if err != nil {
		panic(err)
	}
	conf.RealHW = realhw
	conf.SensorShow = sensorshow
	conf.Configfile = cfile
	log.Printf("%+v\n", conf)

	return &conf
}
