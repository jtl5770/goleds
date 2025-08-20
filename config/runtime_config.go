package config

// RuntimeConfig defines the subset of the configuration that can be
// safely modified at runtime through the web UI. It excludes
// hardware-specific and other sensitive settings.
type RuntimeConfig struct {
	SensorLED    SensorLEDConfig    `yaml:"SensorLED" json:"SensorLED"`
	NightLED     NightLEDConfig     `yaml:"NightLED" json:"NightLED"`
	ClockLED     ClockLEDConfig     `yaml:"ClockLED" json:"ClockLED"`
	AudioLED     AudioLEDConfig     `yaml:"AudioLED" json:"AudioLED"`
	CylonLED     CylonLEDConfig     `yaml:"CylonLED" json:"CylonLED"`
	MultiBlobLED MultiBlobLEDConfig `yaml:"MultiBlobLED" json:"MultiBlobLED"`
}
