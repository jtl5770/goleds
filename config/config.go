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

// validateRGB checks if the RGB slice has exactly 3 components between 0 and 255.
func validateRGB(rgb []float64) error {
	if len(rgb) != 3 {
		return fmt.Errorf("must have exactly 3 components, got %d", len(rgb))
	}
	for i, v := range rgb {
		if v < 0 || v > 255 {
			return fmt.Errorf("component %d must be between 0 and 255: %f", i, v)
		}
	}
	return nil
}

// isValidIndex checks if an index is within the valid range [0, ledsTotal).
func isValidIndex(index, ledsTotal int) bool {
	return index >= 0 && index < ledsTotal
}

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

func (c *SensorLEDConfig) Validate() error {
	if c.RunUpDelay < 0 {
		return fmt.Errorf("RunUpDelay must be non-negative")
	}
	if c.RunDownDelay < 0 {
		return fmt.Errorf("RunDownDelay must be non-negative")
	}
	if c.HoldTime < 0 {
		return fmt.Errorf("HoldTime must be non-negative")
	}
	if err := validateRGB(c.LedRGB); err != nil {
		return fmt.Errorf("LedRGB invalid: %w", err)
	}
	if c.LatchEnabled {
		if c.LatchTriggerValue < 0 || c.LatchTriggerValue > 1023 {
			return fmt.Errorf("LatchTriggerValue must be between 0 and 1023")
		}
		if c.LatchTriggerDelay < 0 {
			return fmt.Errorf("LatchTriggerDelay must be non-negative")
		}
		if c.LatchTime < 0 {
			return fmt.Errorf("LatchTime must be non-negative")
		}
		if err := validateRGB(c.LatchLedRGB); err != nil {
			return fmt.Errorf("LatchLedRGB invalid: %w", err)
		}
	}
	return nil
}

// NightLEDConfig defines the configuration for the NightLED producer.
type NightLEDConfig struct {
	Enabled   bool        `yaml:"Enabled"`
	Latitude  float64     `yaml:"Latitude"`
	Longitude float64     `yaml:"Longitude"`
	LedRGB    [][]float64 `yaml:"LedRGB,flow"`
}

func (c *NightLEDConfig) Validate() error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return fmt.Errorf("Latitude must be between -90 and 90")
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return fmt.Errorf("Longitude must be between -180 and 180")
	}
	for i, rgb := range c.LedRGB {
		if err := validateRGB(rgb); err != nil {
			return fmt.Errorf("LedRGB[%d] invalid: %w", i, err)
		}
	}
	return nil
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

func (c *ClockLEDConfig) Validate(ledsTotal int) error {
	if !isValidIndex(c.StartLedHour, ledsTotal) {
		return fmt.Errorf("StartLedHour out of bounds (0-%d): %d", ledsTotal-1, c.StartLedHour)
	}
	if !isValidIndex(c.EndLedHour, ledsTotal) {
		return fmt.Errorf("EndLedHour out of bounds (0-%d): %d", ledsTotal-1, c.EndLedHour)
	}
	if !isValidIndex(c.StartLedMinute, ledsTotal) {
		return fmt.Errorf("StartLedMinute out of bounds (0-%d): %d", ledsTotal-1, c.StartLedMinute)
	}
	if !isValidIndex(c.EndLedMinute, ledsTotal) {
		return fmt.Errorf("EndLedMinute out of bounds (0-%d): %d", ledsTotal-1, c.EndLedMinute)
	}
	if c.StartLedHour > c.EndLedHour {
		return fmt.Errorf("StartLedHour (%d) > EndLedHour (%d)", c.StartLedHour, c.EndLedHour)
	}
	if c.StartLedMinute > c.EndLedMinute {
		return fmt.Errorf("StartLedMinute (%d) > EndLedMinute (%d)", c.StartLedMinute, c.EndLedMinute)
	}
	if err := validateRGB(c.LedHour); err != nil {
		return fmt.Errorf("LedHour invalid: %w", err)
	}
	if err := validateRGB(c.LedMinute); err != nil {
		return fmt.Errorf("LedMinute invalid: %w", err)
	}
	return nil
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

func (c *AudioLEDConfig) Validate(ledsTotal int) error {
	if !isValidIndex(c.StartLedLeft, ledsTotal) {
		return fmt.Errorf("StartLedLeft out of bounds")
	}
	if !isValidIndex(c.EndLedLeft, ledsTotal) {
		return fmt.Errorf("EndLedLeft out of bounds")
	}
	if !isValidIndex(c.StartLedRight, ledsTotal) {
		return fmt.Errorf("StartLedRight out of bounds")
	}
	if !isValidIndex(c.EndLedRight, ledsTotal) {
		return fmt.Errorf("EndLedRight out of bounds")
	}
	if err := validateRGB(c.LedGreen); err != nil {
		return fmt.Errorf("LedGreen invalid: %w", err)
	}
	if err := validateRGB(c.LedYellow); err != nil {
		return fmt.Errorf("LedYellow invalid: %w", err)
	}
	if err := validateRGB(c.LedRed); err != nil {
		return fmt.Errorf("LedRed invalid: %w", err)
	}
	if c.SampleRate <= 0 {
		return fmt.Errorf("SampleRate must be positive")
	}
	if c.FramesPerBuffer <= 0 {
		return fmt.Errorf("FramesPerBuffer must be positive")
	}
	if c.UpdateFreq < 0 {
		return fmt.Errorf("UpdateFreq must be non-negative")
	}
	if c.MinDB > 0 {
		return fmt.Errorf("MinDB must be <= 0")
	}
	if c.MaxDB > 0 {
		return fmt.Errorf("MaxDB must be <= 0")
	}
	if c.MinDB >= c.MaxDB {
		return fmt.Errorf("MinDB (%f) must be less than MaxDB (%f)", c.MinDB, c.MaxDB)
	}
	return nil
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

func (c *CylonLEDConfig) Validate(ledsTotal int) error {
	if c.Duration < 0 {
		return fmt.Errorf("Duration must be non-negative")
	}
	if c.Delay < 0 {
		return fmt.Errorf("Delay must be non-negative")
	}
	if c.Step <= 0 {
		return fmt.Errorf("Step must be positive")
	}
	if c.Width <= 0 {
		return fmt.Errorf("Width must be positive")
	}
	if ledsTotal > 0 && c.Width > ledsTotal/2 {
		return fmt.Errorf("Width (%d) cannot be larger than half of LedsTotal (%d)", c.Width, ledsTotal)
	}
	if err := validateRGB(c.LedRGB); err != nil {
		return fmt.Errorf("LedRGB invalid: %w", err)
	}
	return nil
}

// MultiBlobLEDConfig defines the configuration for the MultiBlobLED producer.
type MultiBlobLEDConfig struct {
	Enabled  bool          `yaml:"Enabled"`
	Duration time.Duration `yaml:"Duration"`
	Delay    time.Duration `yaml:"Delay"`
	BlobCfg  []BlobCfg     `yaml:"BlobCfg,flow"`
}

func (c *MultiBlobLEDConfig) Validate(ledsTotal int) error {
	if c.Duration < 0 {
		return fmt.Errorf("Duration must be non-negative")
	}
	if c.Delay < 0 {
		return fmt.Errorf("Delay must be non-negative")
	}
	for i, b := range c.BlobCfg {
		if err := b.Validate(ledsTotal); err != nil {
			return fmt.Errorf("BlobCfg[%d] invalid: %w", i, err)
		}
	}
	return nil
}

// BlobCfg defines the configuration for a single blob in the MultiBlobLED producer.
type BlobCfg struct {
	DeltaX float64   `yaml:"DeltaX"`
	X      float64   `yaml:"X"`
	Width  float64   `yaml:"Width"`
	LedRGB []float64 `yaml:"LedRGB,flow"`
}

func (b *BlobCfg) Validate(ledsTotal int) error {
	if b.Width <= 0 {
		return fmt.Errorf("Width must be positive")
	}
	if b.X < 0 || b.X >= float64(ledsTotal) {
		return fmt.Errorf("X (%f) must be between 0 and %d", b.X, ledsTotal-1)
	}
	if err := validateRGB(b.LedRGB); err != nil {
		return fmt.Errorf("LedRGB invalid: %w", err)
	}
	return nil
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
			if !isValidIndex(seg.FirstLed, ledsTotal) || !isValidIndex(seg.LastLed, ledsTotal) {
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
		if !isValidIndex(sensorCfg.LedIndex, ledsTotal) {
			return fmt.Errorf("sensor '%s' has an out-of-bounds LedIndex: %d (LedsTotal=%d)", name, sensorCfg.LedIndex, ledsTotal)
		}
	}

	// 4. Producer Enabled Validation
	if !c.SensorLED.Enabled && !c.NightLED.Enabled && !c.ClockLED.Enabled && !c.AudioLED.Enabled && !c.CylonLED.Enabled && !c.MultiBlobLED.Enabled {
		return fmt.Errorf("at least one producer must be enabled in the configuration")
	}

	if !c.SensorLED.Enabled && (c.MultiBlobLED.Enabled || c.CylonLED.Enabled) {
		return fmt.Errorf("MultiBlobLED and CylonLED producers require the SensorLED producer to be enabled")
	}

	// 5. Producer-Specific Validations
	if c.SensorLED.Enabled {
		if err := c.SensorLED.Validate(); err != nil {
			return fmt.Errorf("SensorLED configuration invalid: %w", err)
		}
	}

	if c.NightLED.Enabled {
		if err := c.NightLED.Validate(); err != nil {
			return fmt.Errorf("NightLED configuration invalid: %w", err)
		}
	}

	if c.ClockLED.Enabled {
		if err := c.ClockLED.Validate(ledsTotal); err != nil {
			return fmt.Errorf("ClockLED configuration invalid: %w", err)
		}
	}

	if c.AudioLED.Enabled {
		if err := c.AudioLED.Validate(ledsTotal); err != nil {
			return fmt.Errorf("AudioLED configuration invalid: %w", err)
		}
	}

	if c.CylonLED.Enabled {
		if err := c.CylonLED.Validate(ledsTotal); err != nil {
			return fmt.Errorf("CylonLED configuration invalid: %w", err)
		}
	}

	if c.MultiBlobLED.Enabled {
		if err := c.MultiBlobLED.Validate(ledsTotal); err != nil {
			return fmt.Errorf("MultiBlobLED configuration invalid: %w", err)
		}
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
