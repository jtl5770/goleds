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
SensorLED:
  Enabled: true
  RunUpDelay: 10ms
  RunDownDelay: 20ms
  HoldTime: 30s
  LedRGB: [255, 0, 0]
`
	err = os.WriteFile(configFile, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("Failed to write dummy config file: %v", err)
	}

	// Call the function to be tested
	conf := ReadConfig(configFile, true, false)

	// Assertions
	assert.True(t, conf.RealHW, "RealHW should be true")
	assert.False(t, conf.SensorShow, "SensorShow should be false")
	assert.Equal(t, configFile, conf.Configfile, "Configfile should be set correctly")

	assert.True(t, conf.SensorLED.Enabled, "SensorLED.Enabled should be true")
	assert.Equal(t, 10*time.Millisecond, conf.SensorLED.RunUpDelay, "SensorLED.RunUpDelay should be 10ms")
	assert.Equal(t, 20*time.Millisecond, conf.SensorLED.RunDownDelay, "SensorLED.RunDownDelay should be 20ms")
	assert.Equal(t, 30*time.Second, conf.SensorLED.HoldTime, "SensorLED.HoldTime should be 30s")
	assert.Equal(t, []float64{255, 0, 0}, conf.SensorLED.LedRGB, "SensorLED.LedRGB should be [255, 0, 0]")
}
