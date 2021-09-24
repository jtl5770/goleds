package producer

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
	Sensors    struct {
		TriggerLeft     int `yaml:"TriggerLeft"`
		TriggerMidLeft  int `yaml:"TriggerMidLeft"`
		TriggerMidRight int `yaml:"TriggerMidRight"`
		TriggerRight    int `yaml:"TriggerRight"`
	} `yaml:"Sensors"`
	SensorLED struct {
		Enabled       bool          `yaml:"Enabled"`
		RunUpMillis   time.Duration `yaml:"RunUpMillis"`
		RunDownMillis time.Duration `yaml:"RunDownMillis"`
		HoldSeconds   time.Duration `yaml:"HoldSeconds"`
		LedRed        byte          `yaml:"LedRed"`
		LedGreen      byte          `yaml:"LedGreen"`
		LedBlue       byte          `yaml:"LedBlue"`
	} `yaml:"SensorLED"`
	NightLED struct {
		Enabled   bool    `yaml:"Enabled"`
		Latitude  float64 `yaml:"Latitude"`
		Longitude float64 `yaml:"Longitude"`
		LedRed    byte    `yaml:"LedRed"`
		LedGreen  byte    `yaml:"LedGreen"`
		LedBlue   byte    `yaml:"LedBlue"`
	} `yaml:"NightLED"`
	HoldLED struct {
		Enabled        bool          `yaml:"Enabled"`
		HoldMinutes    time.Duration `yaml:"HoldMinutes"`
		TriggerSeconds time.Duration `yaml:"TriggerSeconds"`
		LedRed         byte          `yaml:"LedRed"`
		LedGreen       byte          `yaml:"LedGreen"`
		LedBlue        byte          `yaml:"LedBlue"`
	} `yaml:"HoldLED"`
}

func ReadConfig(cfile string, realhw bool) {

	f, err := os.Open(cfile)
	if err != nil {
		log.Fatalf("Can't find config file %s\n%s\n", cfile, err)
		os.Exit(2)
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&CONFIG)
	if err != nil {
		log.Fatalf("Can't decode config file %s\n%s\n", cfile, err)
		os.Exit(2)
	}
	CONFIG.RealHW = realhw
	CONFIG.Configfile = cfile
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
