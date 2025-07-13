package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const CONFILE = "config.yml"

var CONFIG Config

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

// HoldLEDConfig defines the configuration for the HoldLED producer.
type HoldLEDConfig struct {
	Enabled      bool          `yaml:"Enabled"`
	HoldTime     time.Duration `yaml:"HoldTime"`
	TriggerDelay time.Duration `yaml:"TriggerDelay"`
	TriggerValue int           `yaml:"TriggerValue"`
	LedRGB       []float64     `yaml:"LedRGB"`
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
	Enabled  bool          `yaml:"Enabled"`
	Duration time.Duration `yaml:"Duration"`
	Delay    time.Duration `yaml:"Delay"`
	BlobCfg  map[string]struct {
		DeltaX float64   `yaml:"DeltaX"`
		X      float64   `yaml:"X"`
		Width  float64   `yaml:"Width"`
		LedRGB []float64 `yaml:"LedRGB"`
	} `yaml:"BlobCfg"`
}

// HardwareConfig defines the hardware configuration.
type HardwareConfig struct {
	LEDType      string `yaml:"LEDType"`
	SPIFrequency int    `yaml:"SPIFrequency"`
	Display      struct {
		ForceUpdateDelay  time.Duration `yaml:"ForceUpdateDelay"`
		LedsTotal         int           `yaml:"LedsTotal"`
		ColorCorrection   []float64     `yaml:"ColorCorrection"`
		APA102_Brightness byte          `yaml:"APA102_Brightness"`
		LedSegments       map[string][]struct {
			FirstLed     int    `yaml:"FirstLed"`
			LastLed      int    `yaml:"LastLed"`
			SpiMultiplex string `yaml:"SpiMultiplex"`
			Reverse      bool   `yaml:"Reverse"`
		} `yaml:"LedSegments"`
	} `yaml:"Display"`
	Sensors struct {
		SmoothingSize int           `yaml:"SmoothingSize"`
		LoopDelay     time.Duration `yaml:"LoopDelay"`
		SensorCfg     map[string]struct {
			LedIndex     int    `yaml:"LedIndex"`
			SpiMultiplex string `yaml:"SpiMultiplex"`
			AdcChannel   byte   `yaml:"AdcChannel"`
			TriggerValue int    `yaml:"TriggerValue"`
		} `yaml:"SensorCfg"`
	} `yaml:"Sensors"`
	SpiMultiplexGPIO map[string]struct {
		Low  []int `yaml:"Low"`
		High []int `yaml:"High"`
	} `yaml:"SpiMultiplexGPIO"`
}

type Config struct {
	RealHW       bool
	SensorShow   bool
	Configfile   string
	SensorLED    SensorLEDConfig    `yaml:"SensorLED"`
	NightLED     NightLEDConfig     `yaml:"NightLED"`
	HoldLED      HoldLEDConfig      `yaml:"HoldLED"`
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
	CONFIG = conf
	CONFIG.RealHW = realhw
	CONFIG.SensorShow = sensorshow
	CONFIG.Configfile = cfile
	log.Printf("%+v\n", CONFIG)
	return &conf
}
