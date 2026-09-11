package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const commonHardware = `
Hardware:
  WebserverPort: 8080
  LEDType: "ws2801"
  SPIFrequency: 1000000
  Display:
    LedsTotal: 10
    ColorCorrection: [1.0, 1.0, 1.0]
    APA102_Brightness: 31
    LedSegments:
      GroupA:
        - FirstLed: 0
          LastLed: 9
          SpiMultiplex: "L1"
          Reverse: false
  Sensors:
    SmoothingSize: 1
    LoopDelay: 10ms
    Calibration:
      StepDuration: 100ms
      MinMargin: 10
      DeviationFactor: 1.0
      OutlierThreshold: 100
      RetryDelay: 100ms
    SensorCfg:
      S0:
        LedIndex: 0
        SpiMultiplex: "ADC1"
        AdcChannel: 0
  SpiMultiplexGPIO:
    L1:
      Low: [1]
      High: [2]
      CS: 3
    ADC1:
      Low: [1]
      High: [2]
      CS: 3
Logging:
  TUI:
    Level: "DEBUG"
    Format: "text"
    File: "/tmp/goleds-tui.log"
  HW:
    Level: "WARN"
    Format: "json"
    File: "/var/log/goleds-hw.log"
`

const validSensorLED = `
SensorLED:
  Enabled: true
  RunUpDelay: 10ms
  RunDownDelay: 10ms
  HoldTime: 100ms
  LedRGB: [255, 0, 0]
  LatchEnabled: false
  LatchTriggerValue: 500
  LatchTriggerDelay: 1s
  LatchTime: 1s
  LatchLedRGB: [0, 0, 0]
`

const validNightLED = `
NightLED:
  Enabled: false
  Latitude: 0
  Longitude: 0
  LedRGB: [[0, 0, 0]]
`

const validClockLED = `
ClockLED:
  Enabled: false
  StartLedHour: 0
  EndLedHour: 1
  StartLedMinute: 2
  EndLedMinute: 3
  LedHour: [0, 0, 0]
  LedMinute: [0, 0, 0]
`

const validAudioLED = `
AudioLED:
  Enabled: false
  ActiveScene: "Stereo VU"
  Scenes:
    - Name: "Stereo VU"
      Segments:
        - StartLed: 0
          EndLed: 1
          Effect: "LeftVU"
        - StartLed: 2
          EndLed: 3
          Effect: "RightVU"
  VU:
    LedLow: [0, 80, 0]
    LedMid: [40, 40, 0]
    LedHigh: [100, 0, 0]
    SwitchSteps: [60, 80]
    PeakHoldEnabled: true
    PeakHoldTime: 250ms
    PeakDecayRate: 15.0
  Spectrum:
    LedLow: [0, 20, 0]
    LedMid: [50, 35, 0]
    LedHigh: [140, 0, 0]
  UpdateFreq: 10ms
  MinDB: -60
  MaxDB: -10
  Squeezebox:
    Server: "127.0.0.1"
    SlimProtoPort: 3483
    JSONRPCPort: 9000
    PlayerMAC: "00:04:20:11:22:33"
    PlayerName: "Test VU"
    AutoSync: true
    PollInterval: 1500ms
`

const validCylonLED = `
CylonLED:
  Enabled: false
  Duration: 10s
  Delay: 10ms
  Step: 1
  Width: 1
  LedRGB: [0, 0, 0]
`

const validMultiBlobLED = `
MultiBlobLED:
  Enabled: false
  Duration: 10s
  Delay: 10ms
  BlobCfg: []
`

func getBaseConfig() string {
	return commonHardware + validSensorLED + validNightLED + validClockLED + validAudioLED + validCylonLED + validMultiBlobLED
}

func createConfigFile(t *testing.T, content string) string {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yml")
	err := os.WriteFile(configFile, []byte(content), 0666)
	assert.NoError(t, err, "Failed to write config file")
	return configFile
}

func TestReadConfig_Success(t *testing.T) {
	configFile := createConfigFile(t, getBaseConfig())

	conf, err := ReadConfig(configFile)

	assert.NoError(t, err, "ReadConfig should not return an error")
	assert.NotNil(t, conf, "Config should not be nil")
	assert.Equal(t, 10, conf.Hardware.Display.LedsTotal, "Hardware.Display.LedsTotal should be 10")
	assert.True(t, conf.SensorLED.Enabled, "SensorLED.Enabled should be true")
	assert.False(t, conf.NightLED.Enabled, "NightLED.Enabled should be false")
	assert.False(t, conf.ClockLED.Enabled, "ClockLED.Enabled should be false")
	assert.False(t, conf.AudioLED.Enabled, "AudioLED.Enabled should be false")
	assert.False(t, conf.CylonLED.Enabled, "CylonLED.Enabled should be false")
	assert.False(t, conf.MultiBlobLED.Enabled, "MultiBlobLED.Enabled should be false")

	assert.Equal(t, "DEBUG", conf.Logging.TUI.Level, "Logging.TUI.Level should be DEBUG")
	assert.Equal(t, "text", conf.Logging.TUI.Format, "Logging.TUI.Format should be text")
	assert.Equal(t, "/tmp/goleds-tui.log", conf.Logging.TUI.File, "Logging.TUI.File should be /tmp/goleds-tui.log")

	assert.Equal(t, "WARN", conf.Logging.HW.Level, "Logging.HW.Level should be WARN")
	assert.Equal(t, "json", conf.Logging.HW.Format, "Logging.HW.Format should be json")
	assert.Equal(t, "/var/log/goleds-hw.log", conf.Logging.HW.File, "Logging.HW.File should be /var/log/goleds-hw.log")
}

func TestReadConfig_NoProducersEnabled(t *testing.T) {
	// Disable SensorLED (the only one enabled in base config)
	configData := strings.Replace(getBaseConfig(), "Enabled: true", "Enabled: false", 1)
	configFile := createConfigFile(t, configData)

	// Call the function to be tested
	_, err := ReadConfig(configFile)

	// Assertions
	assert.Error(t, err, "ReadConfig should return an error")
	assert.Contains(t, err.Error(), "at least one producer must be enabled", "Error message should indicate that no producers are enabled")
}

func TestReadConfig_AfterProducersWithoutSensorLED(t *testing.T) {
	// Disable SensorLED
	configData := strings.Replace(getBaseConfig(), "Enabled: true", "Enabled: false", 1)
	// Enable CylonLED
	configData = strings.Replace(configData, "CylonLED:\n  Enabled: false", "CylonLED:\n  Enabled: true", 1)

	configFile := createConfigFile(t, configData)

	// Call the function to be tested
	_, err := ReadConfig(configFile)

	// Assertions
	assert.Error(t, err, "ReadConfig should return an error")
	assert.Contains(t, err.Error(), "require the SensorLED producer to be enabled", "Error message should indicate dependency on SensorLED")
}

func TestReadConfig_InvalidRGB(t *testing.T) {
	// Introduce invalid RGB value
	configData := strings.Replace(getBaseConfig(), "[255, 0, 0]", "[256, 0, 0]", 1)
	configFile := createConfigFile(t, configData)

	_, err := ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for RGB > 255")
	assert.Contains(t, err.Error(), "must be between 0 and 255", "Error message should indicate invalid RGB range")
}

func TestReadConfig_InvalidBlobX(t *testing.T) {
	// Add invalid Blob config. MultiBlobLED doesn't need to be enabled for validation to fail now.
	configData := strings.Replace(getBaseConfig(), "BlobCfg: []", "BlobCfg:\n    - { DeltaX: 0.1, X: 11, Width: 1, LedRGB: [0, 0, 0] }", 1)
	configFile := createConfigFile(t, configData)

	_, err := ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for Blob.X >= LedsTotal")
	assert.Contains(t, err.Error(), "must be between 0 and 9", "Error message should indicate Blob.X is out of bounds")
}

func TestReadConfig_InvalidLedSegmentBounds(t *testing.T) {
	// Modify LedSegments to have FirstLed > LastLed
	configData := strings.Replace(getBaseConfig(), "FirstLed: 0\n          LastLed: 9", "FirstLed: 9\n          LastLed: 0", 1)
	configFile := createConfigFile(t, configData)

	_, err := ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for FirstLed > LastLed")
	assert.Contains(t, err.Error(), "has FirstLed > LastLed", "Error message should indicate FirstLed > LastLed")

	// Modify LedSegments to have an out-of-bounds index
	configData = strings.Replace(getBaseConfig(), "LastLed: 9", "LastLed: 10", 1)
	configFile = createConfigFile(t, configData)

	_, err = ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for out-of-bounds segment index")
	assert.Contains(t, err.Error(), "has out-of-bounds indices", "Error message should indicate out-of-bounds segment index")
}

func TestReadConfig_OverlappingLedSegments(t *testing.T) {
	// Add overlapping segments to GroupA
	overlappingSegments := `
      GroupA:
        - FirstLed: 0
          LastLed: 5
          SpiMultiplex: "L1"
        - FirstLed: 4
          LastLed: 9
          SpiMultiplex: "L1"
`
	configData := strings.Replace(getBaseConfig(), "      GroupA:\n        - FirstLed: 0\n          LastLed: 9\n          SpiMultiplex: \"L1\"\n          Reverse: false", overlappingSegments, 1)
	configFile := createConfigFile(t, configData)

	_, err := ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for overlapping segments in the same group")
	assert.Contains(t, err.Error(), "overlapping display segments in group 'GroupA'", "Error message should indicate overlapping segments")
}

func TestReadConfig_InvalidSpiMultiplexKey(t *testing.T) {
	// Change the SpiMultiplex key in a segment to one that isn't defined
	configData := strings.Replace(getBaseConfig(), "SpiMultiplex: \"L1\"", "SpiMultiplex: \"L_NON_EXISTENT\"", 1)
	configFile := createConfigFile(t, configData)

	_, err := ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for undefined SpiMultiplex key in display")
	assert.Contains(t, err.Error(), "uses undefined SpiMultiplex key", "Error message should indicate undefined key")

	// Test for an invalid key in the sensor config
	configData = strings.Replace(getBaseConfig(), "SpiMultiplex: \"ADC1\"", "SpiMultiplex: \"ADC_NON_EXISTENT\"", 1)
	configFile = createConfigFile(t, configData)

	_, err = ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for undefined SpiMultiplex key in sensors")
	assert.Contains(t, err.Error(), "uses undefined SpiMultiplex key", "Error message should indicate undefined key in sensor config")
}

func TestValidateHelperFunctions(t *testing.T) {
	assert.NoError(t, validateRGB([]float64{0, 128, 255}))
	assert.Error(t, validateRGB([]float64{0, 128}))
	assert.Error(t, validateRGB([]float64{0, 128, 255, 100}))
	assert.Error(t, validateRGB([]float64{-1, 128, 255}))
	assert.Error(t, validateRGB([]float64{0, 256, 255}))

	assert.True(t, isValidIndex(0, 10))
	assert.True(t, isValidIndex(9, 10))
	assert.False(t, isValidIndex(-1, 10))
	assert.False(t, isValidIndex(10, 10))
}

func TestAudioEffectType_Parsing(t *testing.T) {
	eff, err := ParseAudioEffectType("LeftVU")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectLeftVU, eff)
	assert.False(t, eff.IsSpectrum())

	eff, err = ParseAudioEffectType("right_vu")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectRightVU, eff)
	assert.False(t, eff.IsSpectrum())

	eff, err = ParseAudioEffectType("MonoVU")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectMonoVU, eff)
	assert.False(t, eff.IsSpectrum())

	eff, err = ParseAudioEffectType("spectrum")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectMonoSpectrum, eff)
	assert.True(t, eff.IsSpectrum())

	eff, err = ParseAudioEffectType("LeftSpectrum")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectLeftSpectrum, eff)
	assert.True(t, eff.IsSpectrum())

	eff, err = ParseAudioEffectType("left_spectrum")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectLeftSpectrum, eff)

	eff, err = ParseAudioEffectType("RightSpectrum")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectRightSpectrum, eff)
	assert.True(t, eff.IsSpectrum())

	eff, err = ParseAudioEffectType("right_spectrum")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectRightSpectrum, eff)

	eff, err = ParseAudioEffectType("MonoSpectrum")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectMonoSpectrum, eff)
	assert.True(t, eff.IsSpectrum())

	eff, err = ParseAudioEffectType("mono_spectrum")
	assert.NoError(t, err)
	assert.Equal(t, AudioEffectMonoSpectrum, eff)

	_, err = ParseAudioEffectType("unknown_effect")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid audio effect")
}

func TestAudioLEDConfig_Validation(t *testing.T) {
	validVU := AudioVUConfig{
		LedLow:          []float64{0, 80, 0},
		LedMid:          []float64{40, 40, 0},
		LedHigh:         []float64{100, 0, 0},
		SwitchSteps:     []int{60, 80},
		PeakHoldEnabled: true,
		PeakHoldTime:    250 * time.Millisecond,
		PeakDecayRate:   15.0,
	}
	validSpectrum := AudioSpectrumConfig{
		LedLow:  []float64{0, 20, 0},
		LedMid:  []float64{50, 35, 0},
		LedHigh: []float64{140, 0, 0},
	}

	// Valid multi-scene config
	audio := AudioLEDConfig{
		Enabled:     true,
		ActiveScene: "Stereo",
		Scenes: []AudioSceneConfig{
			{
				Name: "Stereo",
				Segments: []AudioSegmentConfig{
					{StartLed: 0, EndLed: 9, Effect: AudioEffectLeftVU},
					{StartLed: 10, EndLed: 19, Effect: AudioEffectRightVU},
				},
			},
			{
				Name: "Mono",
				Segments: []AudioSegmentConfig{
					{StartLed: 9, EndLed: 0, Effect: AudioEffectMonoVU},
					{StartLed: 10, EndLed: 19, Effect: AudioEffectMonoVU},
				},
			},
			{
				Name: "Spectrum Scene",
				Segments: []AudioSegmentConfig{
					{StartLed: 0, EndLed: 19, Effect: AudioEffectMonoSpectrum},
				},
			},
			{
				Name: "Stereo Spectrum Scene",
				Segments: []AudioSegmentConfig{
					{StartLed: 0, EndLed: 19, Effect: AudioEffectLeftSpectrum},
					{StartLed: 20, EndLed: 39, Effect: AudioEffectRightSpectrum},
				},
			},
		},
		VU:         validVU,
		Spectrum:   validSpectrum,
		UpdateFreq: 30 * time.Millisecond,
		MinDB:      -60,
		MaxDB:      -3,
		Squeezebox: SqueezeboxConfig{
			Server:        "127.0.0.1",
			SlimProtoPort: 3483,
			JSONRPCPort:   9000,
			PlayerMAC:     "auto",
		},
	}
	assert.NoError(t, audio.Validate(40))

	// Overlapping segments in the same scene must instantly fail
	overlapAudio := audio
	overlapAudio.Scenes = []AudioSceneConfig{
		{
			Name: "Broken Overlap",
			Segments: []AudioSegmentConfig{
				{StartLed: 0, EndLed: 10, Effect: AudioEffectLeftVU},
				{StartLed: 5, EndLed: 15, Effect: AudioEffectRightVU},
			},
		},
	}
	overlapAudio.ActiveScene = "Broken Overlap"
	err := overlapAudio.Validate(40)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "overlapping segments at LED index 5")

	// Spectrum with length < 16 must fail
	for _, specEff := range []AudioEffectType{AudioEffectMonoSpectrum, AudioEffectLeftSpectrum, AudioEffectRightSpectrum} {
		shortSpectrumAudio := audio
		shortSpectrumAudio.Scenes = []AudioSceneConfig{
			{
				Name: "Short Spectrum",
				Segments: []AudioSegmentConfig{
					{StartLed: 0, EndLed: 14, Effect: specEff}, // length = 15
				},
			},
		}
		shortSpectrumAudio.ActiveScene = "Short Spectrum"
		err = shortSpectrumAudio.Validate(40)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "spectrum segment requires at least 16 LEDs")
	}

	// Empty scenes must fail
	noScenesAudio := audio
	noScenesAudio.Scenes = nil
	assert.Error(t, noScenesAudio.Validate(40))

	// Duplicate scene names must fail
	dupSceneAudio := audio
	dupSceneAudio.Scenes = []AudioSceneConfig{
		{Name: "Scene1", Segments: []AudioSegmentConfig{{StartLed: 0, EndLed: 5, Effect: AudioEffectLeftVU}}},
		{Name: "Scene1", Segments: []AudioSegmentConfig{{StartLed: 0, EndLed: 5, Effect: AudioEffectRightVU}}},
	}
	err = dupSceneAudio.Validate(40)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate scene name 'Scene1'")

	// ActiveScene not matching any scene name must fail
	badActiveSceneAudio := audio
	badActiveSceneAudio.ActiveScene = "NonExistent"
	err = badActiveSceneAudio.Validate(40)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ActiveScene 'NonExistent' does not match any configured scene")

	// SwitchSteps validation tests
	badStepsAudio := audio
	badStepsAudio.VU.SwitchSteps = []int{60} // length 1
	assert.Error(t, badStepsAudio.Validate(40))

	badStepsAudio.VU.SwitchSteps = []int{60, 70, 80} // length 3
	assert.Error(t, badStepsAudio.Validate(40))

	badStepsAudio.VU.SwitchSteps = []int{80, 60} // reversed
	assert.Error(t, badStepsAudio.Validate(40))

	badStepsAudio.VU.SwitchSteps = []int{0, 80} // step1 <= 0
	assert.Error(t, badStepsAudio.Validate(40))

	badStepsAudio.VU.SwitchSteps = []int{60, 100} // step2 >= 100
	assert.Error(t, badStepsAudio.Validate(40))

	badStepsAudio.VU.SwitchSteps = nil // nil
	assert.Error(t, badStepsAudio.Validate(40))

	// MinDB >= MaxDB
	badDBAudio := audio
	badDBAudio.MinDB = -10
	badDBAudio.MaxDB = -20
	assert.Error(t, badDBAudio.Validate(40))

	// Invalid Spectrum
	badSpectrumAudio := audio
	badSpectrumAudio.Spectrum.LedLow = []float64{0, 300, 0}
	assert.Error(t, badSpectrumAudio.Validate(40))
}

func TestAudioSpectrumConfig_Validation(t *testing.T) {
	validSpectrum := func() AudioSpectrumConfig {
		return AudioSpectrumConfig{
			LedLow:  []float64{0, 20, 0},
			LedMid:  []float64{50, 35, 0},
			LedHigh: []float64{140, 0, 0},
		}
	}

	tt := t
	tt.Run("valid config", func(t *testing.T) {
		cfg := validSpectrum()
		assert.NoError(t, cfg.Validate())
	})

	tt.Run("invalid LedLow", func(t *testing.T) {
		cfg := validSpectrum()
		cfg.LedLow = []float64{0, 20}
		assert.Error(t, cfg.Validate())
	})
}

func TestSubConfigs_Validation(t *testing.T) {
	// NightLED validation
	night := NightLEDConfig{
		Latitude:  -95,
		Longitude: 0,
		LedRGB:    [][]float64{{0, 0, 0}},
	}
	assert.Error(t, night.Validate())
	night.Latitude = 45
	night.Longitude = 190
	assert.Error(t, night.Validate())
	night.Longitude = 11
	night.LedRGB = [][]float64{{-5, 0, 0}}
	assert.Error(t, night.Validate())

	// ClockLED validation
	clock := ClockLEDConfig{
		StartLedHour:   -1,
		EndLedHour:     10,
		StartLedMinute: 0,
		EndLedMinute:   10,
		LedHour:        []float64{1, 0, 0},
		LedMinute:      []float64{0, 1, 0},
	}
	assert.Error(t, clock.Validate(20))
	clock.StartLedHour = 0
	clock.EndLedHour = 25
	assert.Error(t, clock.Validate(20))

	// CylonLED validation
	cylon := CylonLEDConfig{
		Duration: -1 * time.Second,
		Delay:    10 * time.Millisecond,
		Step:     1,
		Width:    1,
		LedRGB:   []float64{255, 0, 0},
	}
	assert.Error(t, cylon.Validate(20))

	// MultiBlobLED validation
	blob := MultiBlobLEDConfig{
		Duration: -1 * time.Second,
		Delay:    10 * time.Millisecond,
	}
	assert.Error(t, blob.Validate(20))
}
func TestRootConfigFiles(t *testing.T) {
	for _, file := range []string{"../config.yml", "../config.yml.orig"} {
		if _, err := os.Stat(file); err == nil {
			conf, err := ReadConfig(file)
			assert.NoError(t, err, "file %s should parse cleanly", file)
			assert.NotNil(t, conf)
		}
	}
}
