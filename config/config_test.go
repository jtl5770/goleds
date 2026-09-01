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
  Display:
    LedsTotal: 10
  Sensors:
    SmoothingSize: 3
    LoopDelay: 50ms
    Calibration:
      StepDuration: 10s
      MinMargin: 15
      DeviationFactor: 1.5
      OutlierThreshold: 150
      RetryDelay: 5s
    SensorCfg: {}
  SpiMultiplexGPIO: {}
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
  RunDownDelay: 20ms
  HoldTime: 30s
  LedRGB: [255, 0, 0]
  LatchEnabled: false
  LatchTriggerValue: 0
  LatchTriggerDelay: 0s
  LatchTime: 0s
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
  StartLedLeft: 0
  EndLedLeft: 1
  StartLedRight: 2
  EndLedRight: 3
  LedGreen: [0, 0, 0]
  LedYellow: [0, 0, 0]
  LedRed: [0, 0, 0]
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

func createConfigFile(t *testing.T, configData string) string {
	tempDir, err := os.MkdirTemp("", "goleds-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	// We schedule cleanup of the directory, but return the file path
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	configFile := filepath.Join(tempDir, "config.yml")
	err = os.WriteFile(configFile, []byte(configData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write dummy config file: %v", err)
	}
	return configFile
}

func TestReadConfig(t *testing.T) {
	configFile := createConfigFile(t, getBaseConfig())

	// Call the function to be tested
	conf, err := ReadConfig(configFile)
	assert.NoError(t, err, "ReadConfig should not return an error")

	// Assertions
	assert.True(t, conf.SensorLED.Enabled, "SensorLED.Enabled should be true")
	assert.Equal(t, 10*time.Millisecond, conf.SensorLED.RunUpDelay, "SensorLED.RunUpDelay should be 10ms")
	assert.Equal(t, 20*time.Millisecond, conf.SensorLED.RunDownDelay, "SensorLED.RunDownDelay should be 20ms")
	assert.Equal(t, 30*time.Second, conf.SensorLED.HoldTime, "SensorLED.HoldTime should be 30s")
	assert.Equal(t, []float64{255, 0, 0}, conf.SensorLED.LedRGB, "SensorLED.LedRGB should be [255, 0, 0]")

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
	assert.Error(t, err, "ReadConfig should return an error for Blob X out of bounds")
	assert.Contains(t, err.Error(), "must be between 0 and 9", "Error message should indicate invalid X range")
}

func TestSqueezeboxConfig_Validation(t *testing.T) {
	// Valid configuration
	validCfg := SqueezeboxConfig{
		Server:        "192.168.1.100",
		SlimProtoPort: 3483,
		JSONRPCPort:   9000,
		PlayerMAC:     "00:04:20:aa:bb:cc",
		PlayerName:    "Living Room VU",
		AutoSync:      true,
		PollInterval:  1500 * time.Millisecond,
	}
	assert.NoError(t, validCfg.Validate())

	// Empty server
	emptyServerCfg := validCfg
	emptyServerCfg.Server = ""
	assert.Error(t, emptyServerCfg.Validate())
	assert.Contains(t, emptyServerCfg.Validate().Error(), "Server address cannot be empty")

	// Invalid SlimProto port
	badPortCfg := validCfg
	badPortCfg.SlimProtoPort = 70000
	assert.Error(t, badPortCfg.Validate())

	// Invalid MAC
	badMACCfg := validCfg
	badMACCfg.PlayerMAC = "invalid-mac"
	assert.Error(t, badMACCfg.Validate())
	assert.Contains(t, badMACCfg.Validate().Error(), "not a valid MAC address")

	// "auto" MAC is valid
	autoMACCfg := validCfg
	autoMACCfg.PlayerMAC = "auto"
	assert.NoError(t, autoMACCfg.Validate())

	// Negative poll interval
	negPollCfg := validCfg
	negPollCfg.PollInterval = -100 * time.Millisecond
	assert.Error(t, negPollCfg.Validate())
}
