package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "goleds-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a dummy config file
	configFile := filepath.Join(tempDir, "config.yml")
	configData := `
Hardware:
  Display:
    LedsTotal: 10
SensorLED:
  Enabled: true
  RunUpDelay: 10ms
  RunDownDelay: 20ms
  HoldTime: 30s
  LedRGB: [255, 0, 0]
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
	err = os.WriteFile(configFile, []byte(configData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write dummy config file: %v", err)
	}

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
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "goleds-test-no-producers")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a dummy config file with no producers enabled
	configFile := filepath.Join(tempDir, "config.yml")
	configData := `
Hardware:
  Display:
    LedsTotal: 10
SensorLED:
  Enabled: false
NightLED:
  Enabled: false
ClockLED:
  Enabled: false
AudioLED:
  Enabled: false
CylonLED:
  Enabled: false
MultiBlobLED:
  Enabled: false
`
	err = os.WriteFile(configFile, []byte(configData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write dummy config file: %v", err)
	}

	// Call the function to be tested
	_, err = ReadConfig(configFile)

	// Assertions
	assert.Error(t, err, "ReadConfig should return an error")
	assert.Contains(t, err.Error(), "at least one producer must be enabled", "Error message should indicate that no producers are enabled")
}

func TestReadConfig_AfterProducersWithoutSensorLED(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "goleds-test-after-producers")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a dummy config file with after-producers but no sensor producer
	configFile := filepath.Join(tempDir, "config.yml")
	configData := `
Hardware:
  Display:
    LedsTotal: 10
SensorLED:
  Enabled: false
CylonLED:
  Enabled: true
`
	err = os.WriteFile(configFile, []byte(configData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write dummy config file: %v", err)
	}

	// Call the function to be tested
	_, err = ReadConfig(configFile)

	// Assertions
	assert.Error(t, err, "ReadConfig should return an error")
	assert.Contains(t, err.Error(), "require the SensorLED producer to be enabled", "Error message should indicate dependency on SensorLED")
}

func TestReadConfig_InvalidRGB(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "goleds-test-invalid-rgb")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.yml")
	configData := `
Hardware:
  Display:
    LedsTotal: 10
SensorLED:
  Enabled: true
  RunUpDelay: 10ms
  RunDownDelay: 20ms
  HoldTime: 30s
  LedRGB: [256, 0, 0]
`
	err = os.WriteFile(configFile, []byte(configData), 0o644)
	if err != nil {
		t.Fatalf("Failed to write dummy config file: %v", err)
	}

	_, err = ReadConfig(configFile)
	assert.Error(t, err, "ReadConfig should return an error for RGB > 255")
	assert.Contains(t, err.Error(), "must be between 0 and 255", "Error message should indicate invalid RGB range")
}
