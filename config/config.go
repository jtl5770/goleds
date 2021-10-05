package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

const CONFILE = "config.yml"

var CONFIG Config

type Config struct {
	RealHW     bool
	Configfile string
	SensorLED  struct {
		Enabled       bool          `yaml:"Enabled"`
		RunUpMillis   time.Duration `yaml:"RunUpMillis"`
		RunDownMillis time.Duration `yaml:"RunDownMillis"`
		HoldSeconds   time.Duration `yaml:"HoldSeconds"`
		LedRGB        []byte        `yaml:"LedRGB"`
	} `yaml:"SensorLED"`
	NightLED struct {
		Enabled   bool     `yaml:"Enabled"`
		Latitude  float64  `yaml:"Latitude"`
		Longitude float64  `yaml:"Longitude"`
		LedRGB    [][]byte `yaml:"LedRGB"`
	} `yaml:"NightLED"`
	HoldLED struct {
		Enabled        bool          `yaml:"Enabled"`
		HoldMinutes    time.Duration `yaml:"HoldMinutes"`
		TriggerSeconds time.Duration `yaml:"TriggerSeconds"`
		TriggerValue   int           `yaml:"TriggerValue"`
		LedRGB         []byte        `yaml:"LedRGB"`
	} `yaml:"HoldLED"`
	BlobLED struct {
		Enabled bool `yaml:"Enabled"`
		BlobCfg map[string]struct {
			DelayMillis time.Duration `yaml:"DelayMillis"`
			DeltaX      float64       `yaml:"DeltaX"`
			X           float64       `yaml:"X"`
			Width       float64       `yaml:"Width"`
			LedRGB      []byte        `yaml:"LedRGB"`
		} `yaml:"BlobCfg"`
	} `yaml:"BlobLED"`
	Hardware struct {
		Display struct {
			ForceUpdateSeconds time.Duration `yaml:"ForceUpdateSeconds"`
			LedsTotal          int           `yaml:"LedsTotal"`
			SPIFrequency       int           `yaml:"SPIFrequency"`
		} `yaml:"Display"`
		Sensors struct {
			SmoothingSize   int           `yaml:"SmoothingSize"`
			LoopDelayMillis time.Duration `yaml:"LoopDelayMillis"`
			SensorCfg       map[string]struct {
				LedIndex     int  `yaml:"LedIndex"`
				Adc          int  `yaml:"Adc"`
				AdcChannel   byte `yaml:"AdcChannel"`
				TriggerValue int  `yaml:"TriggerValue"`
			} `yaml:"SensorCfg"`
		} `yaml:"Sensors"`
	} `yaml:"Hardware"`
}

func ReadConfig(cfile string, realhw bool) {
	log.Printf("Reading config file %s...", cfile)
	f, err := os.Open(cfile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&CONFIG)
	if err != nil {
		panic(err)
	}
	CONFIG.RealHW = realhw
	CONFIG.Configfile = cfile
	log.Printf("%+v\n", CONFIG)
}

// Local Variables:D
// compile-command: "cd .. && go build"
// End:
