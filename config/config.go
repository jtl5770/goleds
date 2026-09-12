package config

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFile = "config.yml"

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
	Enabled           bool          `yaml:"Enabled" json:"Enabled"`
	RunUpDelay        time.Duration `yaml:"RunUpDelay" json:"RunUpDelay"`
	RunDownDelay      time.Duration `yaml:"RunDownDelay" json:"RunDownDelay"`
	HoldTime          time.Duration `yaml:"HoldTime" json:"HoldTime"`
	LedRGB            []float64     `yaml:"LedRGB,flow" json:"LedRGB"`
	LatchEnabled      bool          `yaml:"LatchEnabled" json:"LatchEnabled"`
	LatchTriggerValue int           `yaml:"LatchTriggerValue" json:"LatchTriggerValue"`
	LatchTriggerDelay time.Duration `yaml:"LatchTriggerDelay" json:"LatchTriggerDelay"`
	LatchTime         time.Duration `yaml:"LatchTime" json:"LatchTime"`
	LatchLedRGB       []float64     `yaml:"LatchLedRGB,flow" json:"LatchLedRGB"`
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

	return nil
}

// NightLEDConfig defines the configuration for the NightLED producer.
type NightLEDConfig struct {
	Enabled   bool        `yaml:"Enabled" json:"Enabled"`
	Latitude  float64     `yaml:"Latitude" json:"Latitude"`
	Longitude float64     `yaml:"Longitude" json:"Longitude"`
	LedRGB    [][]float64 `yaml:"LedRGB,flow" json:"LedRGB"`
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
	Enabled        bool      `yaml:"Enabled" json:"Enabled"`
	StartLedHour   int       `yaml:"StartLedHour" json:"StartLedHour"`
	EndLedHour     int       `yaml:"EndLedHour" json:"EndLedHour"`
	StartLedMinute int       `yaml:"StartLedMinute" json:"StartLedMinute"`
	EndLedMinute   int       `yaml:"EndLedMinute" json:"EndLedMinute"`
	LedHour        []float64 `yaml:"LedHour,flow" json:"LedHour"`
	LedMinute      []float64 `yaml:"LedMinute,flow" json:"LedMinute"`
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

// SqueezeboxConfig holds settings for connecting to Logitech Media Server.
// If Server is empty, modern TLV auto-discovery is used on UDP port 3483,
// and any configured port numbers are ignored.
type SqueezeboxConfig struct {
	Server         string        `yaml:"Server" json:"Server"`
	SlimProtoPort  int           `yaml:"SlimProtoPort" json:"SlimProtoPort"`
	JSONRPCPort    int           `yaml:"JSONRPCPort" json:"JSONRPCPort"`
	PlayerMAC      string        `yaml:"PlayerMAC" json:"PlayerMAC"`
	PlayerName     string        `yaml:"PlayerName" json:"PlayerName"`
	IgnoredPlayers []string      `yaml:"IgnoredPlayers" json:"IgnoredPlayers"`
	AutoSync       bool          `yaml:"AutoSync" json:"AutoSync"`
	PollInterval   time.Duration `yaml:"PollInterval" json:"PollInterval"`
}

func (c *SqueezeboxConfig) Validate() error {
	if c.Server != "" {
		if c.SlimProtoPort < 0 || c.SlimProtoPort > 65535 {
			return fmt.Errorf("SlimProtoPort must be between 0 and 65535, got %d", c.SlimProtoPort)
		}
		if c.JSONRPCPort < 0 || c.JSONRPCPort > 65535 {
			return fmt.Errorf("JSONRPCPort must be between 0 and 65535, got %d", c.JSONRPCPort)
		}
	}
	if c.PlayerMAC != "" && !strings.EqualFold(strings.TrimSpace(c.PlayerMAC), "auto") {
		if _, err := net.ParseMAC(strings.TrimSpace(c.PlayerMAC)); err != nil {
			return fmt.Errorf("PlayerMAC '%s' is not a valid MAC address: %w", c.PlayerMAC, err)
		}
	}
	if c.PollInterval < 0 {
		return fmt.Errorf("PollInterval must be non-negative")
	}
	return nil
}

// AudioEffectType specifies the animation effect rendered on an audio segment.
type AudioEffectType string

const (
	AudioEffectLeftVU        AudioEffectType = "LeftVU"
	AudioEffectRightVU       AudioEffectType = "RightVU"
	AudioEffectMonoVU        AudioEffectType = "MonoVU"
	AudioEffectLeftSpectrum  AudioEffectType = "LeftSpectrum"
	AudioEffectRightSpectrum AudioEffectType = "RightSpectrum"
	AudioEffectMonoSpectrum  AudioEffectType = "MonoSpectrum"
	AudioEffectSpectrum                      = AudioEffectMonoSpectrum // Deprecated alias for AudioEffectMonoSpectrum
)

// IsSpectrum reports whether the effect is any variant of frequency spectrum visualization.
func (e AudioEffectType) IsSpectrum() bool {
	switch e {
	case AudioEffectLeftSpectrum, AudioEffectRightSpectrum, AudioEffectMonoSpectrum:
		return true
	default:
		return false
	}
}

// UnmarshalYAML implements custom unmarshaling to normalize AudioEffectType values.
func (e *AudioEffectType) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := ParseAudioEffectType(s)
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

// ParseAudioEffectType normalizes string inputs into canonical AudioEffectType.
func ParseAudioEffectType(s string) (AudioEffectType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "leftvu", "left_vu", "left":
		return AudioEffectLeftVU, nil
	case "rightvu", "right_vu", "right":
		return AudioEffectRightVU, nil
	case "monovu", "mono_vu", "mono":
		return AudioEffectMonoVU, nil
	case "spectrum", "spec":
		return AudioEffectMonoSpectrum, nil
	case "leftspectrum", "left_spectrum", "leftspec", "left_spec":
		return AudioEffectLeftSpectrum, nil
	case "rightspectrum", "right_spectrum", "rightspec", "right_spec":
		return AudioEffectRightSpectrum, nil
	case "monospectrum", "mono_spectrum", "monospec", "mono_spec":
		return AudioEffectMonoSpectrum, nil
	default:
		return "", fmt.Errorf("invalid audio effect '%s', must be 'LeftVU', 'RightVU', 'MonoVU', 'LeftSpectrum', 'RightSpectrum', or 'MonoSpectrum'", s)
	}
}

// AudioSegmentConfig defines a segment of LEDs assigned to a specific audio effect.
// Direction of animation is determined by comparing StartLed and EndLed:
// - StartLed <= EndLed: forward (from StartLed towards EndLed)
// - StartLed > EndLed: reverse (from StartLed towards EndLed)
type AudioSegmentConfig struct {
	StartLed int             `yaml:"StartLed" json:"StartLed"`
	EndLed   int             `yaml:"EndLed" json:"EndLed"`
	Effect   AudioEffectType `yaml:"Effect" json:"Effect"`
}

func (s *AudioSegmentConfig) Validate(ledsTotal int) error {
	if !isValidIndex(s.StartLed, ledsTotal) {
		return fmt.Errorf("StartLed out of bounds (0-%d): %d", ledsTotal-1, s.StartLed)
	}
	if !isValidIndex(s.EndLed, ledsTotal) {
		return fmt.Errorf("EndLed out of bounds (0-%d): %d", ledsTotal-1, s.EndLed)
	}
	if _, err := ParseAudioEffectType(string(s.Effect)); err != nil {
		return err
	}

	length := max(s.StartLed, s.EndLed) - min(s.StartLed, s.EndLed) + 1
	if s.Effect.IsSpectrum() && length < 16 {
		return fmt.Errorf("spectrum segment requires at least 16 LEDs, got %d (start=%d, end=%d)", length, s.StartLed, s.EndLed)
	}
	return nil
}

// AudioSceneConfig defines a named collection of LED segments and their assigned effects.
type AudioSceneConfig struct {
	Name     string               `yaml:"Name" json:"Name"`
	Segments []AudioSegmentConfig `yaml:"Segments" json:"Segments"`
}

func (sc *AudioSceneConfig) Validate(ledsTotal int) error {
	name := strings.TrimSpace(sc.Name)
	if name == "" {
		return fmt.Errorf("scene name cannot be empty")
	}
	if len(sc.Segments) == 0 {
		return fmt.Errorf("scene '%s' must contain at least one segment", sc.Name)
	}

	occupied := make([]bool, ledsTotal)
	for i := range sc.Segments {
		seg := &sc.Segments[i]
		if err := seg.Validate(ledsTotal); err != nil {
			return fmt.Errorf("scene '%s' segment %d invalid: %w", sc.Name, i, err)
		}

		start := min(seg.StartLed, seg.EndLed)
		end := max(seg.StartLed, seg.EndLed)
		for idx := start; idx <= end; idx++ {
			if occupied[idx] {
				return fmt.Errorf("scene '%s' has overlapping segments at LED index %d", sc.Name, idx)
			}
			occupied[idx] = true
		}
	}
	return nil
}

// AudioVUConfig isolates VU-meter styling and peak indicator parameters.
type AudioVUConfig struct {
	LedLow          []float64     `yaml:"LedLow,flow" json:"LedLow"`
	LedMid          []float64     `yaml:"LedMid,flow" json:"LedMid"`
	LedHigh         []float64     `yaml:"LedHigh,flow" json:"LedHigh"`
	SwitchSteps     []int         `yaml:"SwitchSteps,flow" json:"SwitchSteps"`
	PeakHoldEnabled bool          `yaml:"PeakHoldEnabled" json:"PeakHoldEnabled"`
	PeakHoldTime    time.Duration `yaml:"PeakHoldTime" json:"PeakHoldTime"`
	PeakDecayRate   float64       `yaml:"PeakDecayRate" json:"PeakDecayRate"`
}

func (c *AudioVUConfig) Validate() error {
	if err := validateRGB(c.LedLow); err != nil {
		return fmt.Errorf("LedLow invalid: %w", err)
	}
	if err := validateRGB(c.LedMid); err != nil {
		return fmt.Errorf("LedMid invalid: %w", err)
	}
	if err := validateRGB(c.LedHigh); err != nil {
		return fmt.Errorf("LedHigh invalid: %w", err)
	}
	if len(c.SwitchSteps) != 2 {
		return fmt.Errorf("SwitchSteps must contain exactly 2 percentages, got %d", len(c.SwitchSteps))
	}
	if c.SwitchSteps[0] <= 0 || c.SwitchSteps[0] >= c.SwitchSteps[1] || c.SwitchSteps[1] >= 100 {
		return fmt.Errorf("SwitchSteps must satisfy 0 < step1 (%d) < step2 (%d) < 100", c.SwitchSteps[0], c.SwitchSteps[1])
	}
	if c.PeakHoldTime < 0 {
		return fmt.Errorf("PeakHoldTime must be non-negative")
	}
	if c.PeakDecayRate < 0 {
		return fmt.Errorf("PeakDecayRate must be non-negative")
	}
	return nil
}

// AudioSpectrumConfig isolates Spectrum analyzer styling and gradient parameters.
type AudioSpectrumConfig struct {
	LedLow  []float64 `yaml:"LedLow,flow" json:"LedLow"`
	LedMid  []float64 `yaml:"LedMid,flow" json:"LedMid"`
	LedHigh []float64 `yaml:"LedHigh,flow" json:"LedHigh"`
}

func (c *AudioSpectrumConfig) Validate() error {
	if err := validateRGB(c.LedLow); err != nil {
		return fmt.Errorf("LedLow invalid: %w", err)
	}
	if err := validateRGB(c.LedMid); err != nil {
		return fmt.Errorf("LedMid invalid: %w", err)
	}
	if err := validateRGB(c.LedHigh); err != nil {
		return fmt.Errorf("LedHigh invalid: %w", err)
	}
	return nil
}

// AudioLEDConfig defines the configuration for the AudioLED producer.
type AudioLEDConfig struct {
	Enabled     bool                `yaml:"Enabled" json:"Enabled"`
	ActiveScene string              `yaml:"ActiveScene,omitempty" json:"ActiveScene,omitempty"`
	Scenes      []AudioSceneConfig  `yaml:"Scenes" json:"Scenes"`
	VU          AudioVUConfig       `yaml:"VU" json:"VU"`
	Spectrum    AudioSpectrumConfig `yaml:"Spectrum" json:"Spectrum"`
	UpdateFreq  time.Duration       `yaml:"UpdateFreq" json:"UpdateFreq"`
	MinDB       float64             `yaml:"MinDB" json:"MinDB"`
	MaxDB       float64             `yaml:"MaxDB" json:"MaxDB"`
	Squeezebox  SqueezeboxConfig    `yaml:"Squeezebox" json:"Squeezebox"`
}

func (c *AudioLEDConfig) Validate(ledsTotal int) error {
	if len(c.Scenes) == 0 {
		return fmt.Errorf("AudioLED must define at least one scene in 'Scenes'")
	}

	sceneNames := make(map[string]struct{}, len(c.Scenes))
	for i := range c.Scenes {
		if err := c.Scenes[i].Validate(ledsTotal); err != nil {
			return fmt.Errorf("Scenes[%d] invalid: %w", i, err)
		}
		if _, exists := sceneNames[c.Scenes[i].Name]; exists {
			return fmt.Errorf("duplicate scene name '%s' in AudioLED.Scenes", c.Scenes[i].Name)
		}
		sceneNames[c.Scenes[i].Name] = struct{}{}
	}

	if c.ActiveScene != "" {
		if _, ok := sceneNames[c.ActiveScene]; !ok {
			return fmt.Errorf("ActiveScene '%s' does not match any configured scene", c.ActiveScene)
		}
	}

	if err := c.VU.Validate(); err != nil {
		return fmt.Errorf("AudioLED VU config invalid: %w", err)
	}
	if err := c.Spectrum.Validate(); err != nil {
		return fmt.Errorf("AudioLED Spectrum config invalid: %w", err)
	}

	if c.UpdateFreq <= 0 {
		return fmt.Errorf("UpdateFreq must be positive")
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
	if c.Enabled {
		if err := c.Squeezebox.Validate(); err != nil {
			return fmt.Errorf("Squeezebox configuration invalid: %w", err)
		}
	}
	return nil
}

// CylonLEDConfig defines the configuration for the CylonLED producer.
type CylonLEDConfig struct {
	Enabled  bool          `yaml:"Enabled" json:"Enabled"`
	Duration time.Duration `yaml:"Duration" json:"Duration"`
	Delay    time.Duration `yaml:"Delay" json:"Delay"`
	Step     float64       `yaml:"Step" json:"Step"`
	Width    int           `yaml:"Width" json:"Width"`
	LedRGB   []float64     `yaml:"LedRGB,flow" json:"LedRGB"`
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
	Enabled  bool          `yaml:"Enabled" json:"Enabled"`
	Duration time.Duration `yaml:"Duration" json:"Duration"`
	Delay    time.Duration `yaml:"Delay" json:"Delay"`
	BlobCfg  []BlobCfg     `yaml:"BlobCfg" json:"BlobCfg"`
}

func (c *MultiBlobLEDConfig) Validate(ledsTotal int) error {
	if c.Duration < 0 {
		return fmt.Errorf("Duration must be non-negative")
	}
	if c.Delay < 0 {
		return fmt.Errorf("Delay must be positive")
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
	DeltaX float64   `yaml:"DeltaX" json:"DeltaX"`
	X      float64   `yaml:"X" json:"X"`
	Width  float64   `yaml:"Width" json:"Width"`
	LedRGB []float64 `yaml:"LedRGB,flow" json:"LedRGB"`
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
}

// CalibrationConfig defines automatic calibration parameters for IR sensors.
type CalibrationConfig struct {
	StepDuration     time.Duration `yaml:"StepDuration"`
	MinMargin        int           `yaml:"MinMargin"`
	DeviationFactor  float64       `yaml:"DeviationFactor"`
	OutlierThreshold int           `yaml:"OutlierThreshold"`
	RetryDelay       time.Duration `yaml:"RetryDelay"`
}

func (c *CalibrationConfig) Validate() error {
	if c.StepDuration <= 0 {
		return fmt.Errorf("StepDuration must be positive")
	}
	if c.MinMargin < 0 {
		return fmt.Errorf("MinMargin must be non-negative")
	}
	if c.DeviationFactor < 0 {
		return fmt.Errorf("DeviationFactor must be non-negative")
	}
	if c.OutlierThreshold <= 0 {
		return fmt.Errorf("OutlierThreshold must be positive")
	}
	if c.RetryDelay < 0 {
		return fmt.Errorf("RetryDelay must be non-negative")
	}
	return nil
}

// SensorsConfig defines the sensors configuration.
type SensorsConfig struct {
	SmoothingSize int                  `yaml:"SmoothingSize"`
	LoopDelay     time.Duration        `yaml:"LoopDelay"`
	Calibration   CalibrationConfig    `yaml:"Calibration"`
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
	if len(c.Hardware.Sensors.SensorCfg) > 0 {
		if c.Hardware.Sensors.SmoothingSize <= 0 {
			return fmt.Errorf("Sensors SmoothingSize must be positive")
		}
		if c.Hardware.Sensors.LoopDelay <= 0 {
			return fmt.Errorf("Sensors LoopDelay must be positive")
		}
		if err := c.Hardware.Sensors.Calibration.Validate(); err != nil {
			return fmt.Errorf("Sensors Calibration configuration invalid: %w", err)
		}
		for name, sensorCfg := range c.Hardware.Sensors.SensorCfg {
			if !isValidIndex(sensorCfg.LedIndex, ledsTotal) {
				return fmt.Errorf("sensor '%s' has an out-of-bounds LedIndex: %d (LedsTotal=%d)", name, sensorCfg.LedIndex, ledsTotal)
			}
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
	if err := c.SensorLED.Validate(); err != nil {
		return fmt.Errorf("SensorLED configuration invalid: %w", err)
	}

	if err := c.NightLED.Validate(); err != nil {
		return fmt.Errorf("NightLED configuration invalid: %w", err)
	}

	if err := c.ClockLED.Validate(ledsTotal); err != nil {
		return fmt.Errorf("ClockLED configuration invalid: %w", err)
	}

	if err := c.AudioLED.Validate(ledsTotal); err != nil {
		return fmt.Errorf("AudioLED configuration invalid: %w", err)
	}

	if err := c.CylonLED.Validate(ledsTotal); err != nil {
		return fmt.Errorf("CylonLED configuration invalid: %w", err)
	}

	if err := c.MultiBlobLED.Validate(ledsTotal); err != nil {
		return fmt.Errorf("MultiBlobLED configuration invalid: %w", err)
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

	data, err := os.ReadFile(cfile)
	if err != nil {
		slog.Error("Can't open config file", "file", cfile, "error", err)
		return nil, err
	}
	if err := yaml.Unmarshal(data, &conf); err != nil {
		slog.Error("Can't parse config file", "file", cfile, "error", err)
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		slog.Error("Config validation failed", "error", err)
		return nil, err
	}

	return &conf, nil
}
