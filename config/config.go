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
		Enabled      bool          `yaml:"Enabled"`
		RunUpDelay   time.Duration `yaml:"RunUpDelay"`
		RunDownDelay time.Duration `yaml:"RunDownDelay"`
		HoldTime     time.Duration `yaml:"HoldTime"`
		LedRGB       []float64     `yaml:"LedRGB"`
	} `yaml:"SensorLED"`
	NightLED struct {
		Enabled   bool        `yaml:"Enabled"`
		Latitude  float64     `yaml:"Latitude"`
		Longitude float64     `yaml:"Longitude"`
		LedRGB    [][]float64 `yaml:"LedRGB"`
	} `yaml:"NightLED"`
	HoldLED struct {
		Enabled      bool          `yaml:"Enabled"`
		HoldTime     time.Duration `yaml:"HoldTime"`
		TriggerDelay time.Duration `yaml:"TriggerDelay"`
		TriggerValue int           `yaml:"TriggerValue"`
		LedRGB       []float64     `yaml:"LedRGB"`
	} `yaml:"HoldLED"`
	CylonLED struct {
		Enabled  bool          `yaml:"Enabled"`
		Duration time.Duration `yaml:"Duration"`
		Delay    time.Duration `yaml:"Delay"`
		Step     float64       `yaml:"Step"`
		Width    int           `yaml:"Width"`
		LedRGB   []float64     `yaml:"LedRGB"`
	} `yaml:"CylonLED"`
	MultiBlobLED struct {
		Enabled  bool          `yaml:"Enabled"`
		Trigger  bool          `yaml:"Trigger"`
		Duration time.Duration `yaml:"Duration"`
		Delay    time.Duration `yaml:"Delay"`
		BlobCfg  map[string]struct {
			DeltaX float64   `yaml:"DeltaX"`
			X      float64   `yaml:"X"`
			Width  float64   `yaml:"Width"`
			LedRGB []float64 `yaml:"LedRGB"`
		} `yaml:"BlobCfg"`
	} `yaml:"MultiBlobLED"`
	Hardware struct {
		SPIFrequency int `yaml:"SPIFrequency"`
		Display      struct {
			ForceUpdateDelay time.Duration `yaml:"ForceUpdateDelay"`
			LedsTotal        int           `yaml:"LedsTotal"`
			ColorCorrection  []float64     `yaml:"ColorCorrection"`
			Segments         []struct {
				FirstLed int  `yaml:"FirstLed"`
				LastLed  int  `yaml:"LastLed"`
				Visible  bool `yaml:"Visible"`
			} `yaml:"Segments"`
		} `yaml:"Display"`
		Sensors struct {
			SmoothingSize int           `yaml:"SmoothingSize"`
			LoopDelay     time.Duration `yaml:"LoopDelay"`
			SensorCfg     map[string]struct {
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
	var conf Config
	err = decoder.Decode(&conf)
	if err != nil {
		panic(err)
	}
	CONFIG = conf
	CONFIG.RealHW = realhw
	CONFIG.Configfile = cfile
	log.Printf("%+v\n", CONFIG)
}

// Local Variables:D
// compile-command: "cd .. && go build"
// End:
