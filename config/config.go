package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const CONFILE = "config.yml"

// SensorLEDConfig defines the configuration for the SensorLED producer.
type SensorLEDConfig struct {
	Enabled           bool          `yaml:"Enabled"`
	RunUpDelay        time.Duration `yaml:"RunUpDelay"`
	RunDownDelay      time.Duration `yaml:"RunDownDelay"`
	HoldTime          time.Duration `yaml:"HoldTime"`
	LedRGB            []float64     `yaml:"LedRGB,flow"`
	LatchEnabled      bool          `yaml:"LatchEnabled"`
	LatchTriggerValue int           `yaml:"LatchTriggerValue"`
	LatchTriggerDelay time.Duration `yaml:"LatchTriggerDelay"`
	LatchTime         time.Duration `yaml:"LatchTime"`
	LatchLedRGB       []float64     `yaml:"LatchLedRGB,flow"`
}

// NightLEDConfig defines the configuration for the NightLED producer.
type NightLEDConfig struct {
	Enabled   bool        `yaml:"Enabled"`
	Latitude  float64     `yaml:"Latitude"`
	Longitude float64     `yaml:"Longitude"`
	LedRGB    [][]float64 `yaml:"LedRGB,flow"`
}

// ClockLEDConfig defines the configuration for the ClockLED producer.
type ClockLEDConfig struct {
	Enabled        bool      `yaml:"Enabled"`
	StartLedHour   int       `yaml:"StartLedHour"`
	EndLedHour     int       `yaml:"EndLedHour"`
	StartLedMinute int       `yaml:"StartLedMinute"`
	EndLedMinute   int       `yaml:"EndLedMinute"`
	LedHour        []float64 `yaml:"LedHour,flow"`
	LedMinute      []float64 `yaml:"LedMinute,flow"`
}

// AudioLEDConfig defines the configuration for the AudioLED producer.
type AudioLEDConfig struct {
	Enabled         bool          `yaml:"Enabled"`
	Device          string        `yaml:"Device"`
	StartLedLeft    int           `yaml:"StartLedLeft"`
	EndLedLeft      int           `yaml:"EndLedLeft"`
	StartLedRight   int           `yaml:"StartLedRight"`
	EndLedRight     int           `yaml:"EndLedRight"`
	LedGreen        []float64     `yaml:"LedGreen,flow"`
	LedYellow       []float64     `yaml:"LedYellow,flow"`
	LedRed          []float64     `yaml:"LedRed,flow"`
	SampleRate      int           `yaml:"SampleRate"`
	FramesPerBuffer int           `yaml:"FramesPerBuffer"`
	UpdateFreq      time.Duration `yaml:"UpdateFreq"`
	MinDB           float64       `yaml:"MinDB"`
	MaxDB           float64       `yaml:"MaxDB"`
}

// CylonLEDConfig defines the configuration for the CylonLED producer.
type CylonLEDConfig struct {
	Enabled  bool          `yaml:"Enabled"`
	Duration time.Duration `yaml:"Duration"`
	Delay    time.Duration `yaml:"Delay"`
	Step     float64       `yaml:"Step"`
	Width    int           `yaml:"Width"`
	LedRGB   []float64     `yaml:"LedRGB,flow"`
}

// MultiBlobLEDConfig defines the configuration for the MultiBlobLED producer.
type MultiBlobLEDConfig struct {
	Enabled  bool          `yaml:"Enabled"`
	Duration time.Duration `yaml:"Duration"`
	Delay    time.Duration `yaml:"Delay"`
	BlobCfg  []BlobCfg     `yaml:"BlobCfg,flow"`
}

// BlobCfg defines the configuration for a single blob in the MultiBlobLED producer.
type BlobCfg struct {
	DeltaX float64   `yaml:"DeltaX"`
	X      float64   `yaml:"X"`
	Width  float64   `yaml:"Width"`
	LedRGB []float64 `yaml:"LedRGB,flow"`
}

// HardwareConfig defines the hardware configuration.
type HardwareConfig struct {
	WebserverPort    uint16        `yaml:"WebserverPort"`
	LEDType          string        `yaml:"LEDType"`
	SPIFrequency     int           `yaml:"SPIFrequency"`
	Display          DisplayConfig `yaml:"Display"`
	Sensors          SensorsConfig `yaml:"Sensors"`
	SpiMultiplexGPIO map[string]struct {
		Low  []int `yaml:"Low,flow"`
		High []int `yaml:"High,flow"`
		CS   int   `yaml:"CS,flow"`
	} `yaml:"SpiMultiplexGPIO"`
}

// DisplayConfig defines the display configuration.
type DisplayConfig struct {
	ForceUpdateDelay  time.Duration                 `yaml:"ForceUpdateDelay"`
	LedsTotal         int                           `yaml:"LedsTotal"`
	ColorCorrection   []float64                     `yaml:"ColorCorrection,flow"`
	APA102_Brightness byte                          `yaml:"APA102_Brightness"`
	LedSegments       map[string][]LedSegmentConfig `yaml:"LedSegments,flow"`
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

type SingleLoggingConfig struct {
	Level  string `yaml:"Level"`
	Format string `yaml:"Format"`
	File   string `yaml:"File"`
}

type LoggingConfig struct {
	TUI SingleLoggingConfig `yaml:"TUI"`
	HW  SingleLoggingConfig `yaml:"HW"`
}

type Config struct {
	SensorLED    SensorLEDConfig    `yaml:"SensorLED"`
	NightLED     NightLEDConfig     `yaml:"NightLED"`
	ClockLED     ClockLEDConfig     `yaml:"ClockLED"`
	AudioLED     AudioLEDConfig     `yaml:"AudioLED"`
	CylonLED     CylonLEDConfig     `yaml:"CylonLED"`
	MultiBlobLED MultiBlobLEDConfig `yaml:"MultiBlobLED"`
	Hardware     HardwareConfig     `yaml:"Hardware"`
	Logging      LoggingConfig      `yaml:"Logging"`
}

// Validate performs a comprehensive sanity check of the configuration.
func (c *Config) Validate() error {
	// General validation for LED indices
	ledsTotal := c.Hardware.Display.LedsTotal
	if ledsTotal <= 0 {
		return fmt.Errorf("LedsTotal must be a positive number")
	}

	// Helper function to check if an index is within the valid range
	isValidIndex := func(index int) bool {
		return index >= 0 && index < ledsTotal
	}

	// 1. SPI Multiplexer Validation (only in hardware mode)
	// Check sensors
	for name, sensorCfg := range c.Hardware.Sensors.SensorCfg {
		if _, ok := c.Hardware.SpiMultiplexGPIO[sensorCfg.SpiMultiplex]; !ok {
			return fmt.Errorf("sensor '%s' uses undefined SpiMultiplex key: '%s'", name, sensorCfg.SpiMultiplex)
		}
	}
	// Check LED segments
	for groupName, segments := range c.Hardware.Display.LedSegments {
		for i, segmentCfg := range segments {
			if _, ok := c.Hardware.SpiMultiplexGPIO[segmentCfg.SpiMultiplex]; !ok {
				return fmt.Errorf("LED segment %d in group '%s' uses undefined SpiMultiplex key: '%s'", i, groupName, segmentCfg.SpiMultiplex)
			}
		}
	}

	// 2. LED Segment Validation
	for name, segArray := range c.Hardware.Display.LedSegments {
		allLeds := make([]bool, ledsTotal)
		for _, seg := range segArray {
			if !isValidIndex(seg.FirstLed) || !isValidIndex(seg.LastLed) {
				return fmt.Errorf("segment in group '%s' has out-of-bounds indices: FirstLed=%d, LastLed=%d (LedsTotal=%d)", name, seg.FirstLed, seg.LastLed, ledsTotal)
			}
			if seg.FirstLed > seg.LastLed {
				return fmt.Errorf("segment in group '%s' has FirstLed > LastLed", name)
			}
			for i := seg.FirstLed; i <= seg.LastLed; i++ {
				if allLeds[i] {
					return fmt.Errorf("overlapping display segments in group '%s' at index %d", name, i)
				}
				allLeds[i] = true
			}
		}
	}

	// 3. Sensor Configuration Validation
	for name, sensorCfg := range c.Hardware.Sensors.SensorCfg {
		if !isValidIndex(sensorCfg.LedIndex) {
			return fmt.Errorf("sensor '%s' has an out-of-bounds LedIndex: %d (LedsTotal=%d)", name, sensorCfg.LedIndex, ledsTotal)
		}
	}

	// 4. Producer-Specific Validations
	if c.ClockLED.Enabled {
		clk := c.ClockLED
		if !isValidIndex(clk.StartLedHour) || !isValidIndex(clk.EndLedHour) || !isValidIndex(clk.StartLedMinute) || !isValidIndex(clk.EndLedMinute) {
			return fmt.Errorf("ClockLED configuration has out-of-bounds LED indices")
		}
		if clk.StartLedHour > clk.EndLedHour || clk.StartLedMinute > clk.EndLedMinute {
			return fmt.Errorf("ClockLED configuration has start index greater than end index")
		}
	}

	if c.AudioLED.Enabled {
		aud := c.AudioLED
		if !isValidIndex(aud.StartLedLeft) || !isValidIndex(aud.EndLedLeft) || !isValidIndex(aud.StartLedRight) || !isValidIndex(aud.EndLedRight) {
			return fmt.Errorf("AudioLED configuration has out-of-bounds LED indices")
		}
	}

	// 5. Producer Enabled Validation
	if !c.SensorLED.Enabled && !c.NightLED.Enabled && !c.ClockLED.Enabled && !c.AudioLED.Enabled && !c.CylonLED.Enabled && !c.MultiBlobLED.Enabled {
		return fmt.Errorf("at least one producer must be enabled in the configuration")
	}

	if !c.SensorLED.Enabled && (c.MultiBlobLED.Enabled || c.CylonLED.Enabled) {
		return fmt.Errorf("MultiBlobLED and CylonLED producers require the SensorLED producer to be enabled")
	}

	return nil
}

func ReadConfig(cfile string) (*Config, error) {
	slog.Info("Reading config file", "file", cfile)
	var conf Config
	exe, err := os.Executable()
	if err != nil {
		slog.Error("Path of current executable can't be found", "error", err)
		return nil, err
	}
	exPath := filepath.Dir(exe)
	oldcfile := exPath + "/config.yml.orig"

	if _, err := os.Stat(cfile); err != nil && os.IsNotExist(err) {
		slog.Warn("Config file does not exist, using default values to create it.", "config file", cfile)

		in, err := os.Open(oldcfile)
		if err != nil {
			slog.Error("Default config file 'config.yml.orig' can't be found. Your installation is incomplete.")
			return nil, err
		}
		out, err := os.Create(cfile)
		if err != nil {
			slog.Error("Can't create config file", "file", cfile)
			return nil, err
		}
		defer func() {
			in.Close()
			if cerr := out.Close(); cerr != nil {
				slog.Error("Error closing new config file", "file", cfile, "error", cerr)
				if err == nil {
					err = cerr
				}
			}
		}()
		if _, err := io.Copy(out, in); err != nil {
			slog.Error("Error copying config.yml.orig to new config file", "file", cfile, "error", err)
			return nil, err
		}
	} else {
		// f, err := os.Open(oldcfile)
		// if err != nil {
		// 	return nil, err
		// }
		// decoder := yaml.NewDecoder(f)
		// err = decoder.Decode(&conf)
		// if err != nil {
		// 	return nil, err
		// }
	}

	f, err := os.Open(cfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&conf)
	if err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	slog.Debug("Read config", "config", conf)
	return &conf, nil
}
