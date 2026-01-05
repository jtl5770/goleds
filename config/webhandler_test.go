package config

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func getValidRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		LedsTotal: 100,
		SensorLED: SensorLEDConfig{
			Enabled:           true,
			RunUpDelay:        10 * time.Millisecond,
			RunDownDelay:      20 * time.Millisecond,
			HoldTime:          5 * time.Second,
			LedRGB:            []float64{0, 0, 0},
			LatchEnabled:      false,
			LatchTriggerValue: 0,
			LatchTriggerDelay: 0,
			LatchTime:         0,
			LatchLedRGB:       []float64{0, 0, 0},
		},
		NightLED: NightLEDConfig{
			Enabled:   false,
			Latitude:  0,
			Longitude: 0,
			LedRGB:    [][]float64{{0, 0, 0}},
		},
		ClockLED: ClockLEDConfig{
			Enabled:        false,
			StartLedHour:   0,
			EndLedHour:     1,
			StartLedMinute: 2,
			EndLedMinute:   3,
			LedHour:        []float64{0, 0, 0},
			LedMinute:      []float64{0, 0, 0},
		},
		AudioLED: AudioLEDConfig{
			Enabled:         false,
			Device:          "default",
			StartLedLeft:    0,
			EndLedLeft:      1,
			StartLedRight:   2,
			EndLedRight:     3,
			LedGreen:        []float64{0, 0, 0},
			LedYellow:       []float64{0, 0, 0},
			LedRed:          []float64{0, 0, 0},
			SampleRate:      44100,
			FramesPerBuffer: 1024,
			UpdateFreq:      10 * time.Millisecond,
			MinDB:           -60,
			MaxDB:           -10,
		},
		CylonLED: CylonLEDConfig{
			Enabled:  false,
			Duration: 10 * time.Second,
			Delay:    10 * time.Millisecond,
			Step:     1,
			Width:    1,
			LedRGB:   []float64{0, 0, 0},
		},
		MultiBlobLED: MultiBlobLEDConfig{
			Enabled:  false,
			Duration: 10 * time.Second,
			Delay:    10 * time.Millisecond,
			BlobCfg:  []BlobCfg{},
		},
	}
}

func TestConfigHandler_SetValidation(t *testing.T) {
	// 1. Setup temporary environment
	tempDir, err := os.MkdirTemp("", "goleds-webtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.yml")

	// Create a valid initial configuration
	baseRuntime := getValidRuntimeConfig()
	initialConfig := Config{
		Hardware: HardwareConfig{
			Display: DisplayConfig{LedsTotal: 100},
			Sensors: SensorsConfig{SensorCfg: map[string]SensorCfg{}},
			SpiMultiplexGPIO: map[string]struct {
				Low  []int `yaml:"Low,flow"`
				High []int `yaml:"High,flow"`
				CS   int   `yaml:"CS,flow"`
			}{},
		},
		SensorLED:    baseRuntime.SensorLED,
		NightLED:     baseRuntime.NightLED,
		ClockLED:     baseRuntime.ClockLED,
		AudioLED:     baseRuntime.AudioLED,
		CylonLED:     baseRuntime.CylonLED,
		MultiBlobLED: baseRuntime.MultiBlobLED,
	}

	// We need to write this as proper YAML first
	data, _ := yaml.Marshal(initialConfig)
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// 2. Define Test Cases
	tests := []struct {
		name         string
		payload      RuntimeConfig
		wantStatus   int
		wantErrorMsg string
		shouldModify bool
	}{
		{
			name: "Valid Update",
			payload: func() RuntimeConfig {
				c := getValidRuntimeConfig()
				c.SensorLED.LedRGB = []float64{50, 50, 50}
				c.SensorLED.HoldTime = 10 * time.Second
				return c
			}(),
			wantStatus:   http.StatusOK,
			shouldModify: true,
		},
		{
			name: "Invalid RGB (>255)",
			payload: func() RuntimeConfig {
				c := getValidRuntimeConfig()
				c.SensorLED.LedRGB = []float64{300, 0, 0}
				return c
			}(),
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be between 0 and 255",
			shouldModify: false,
		},
		{
			name: "Invalid RGB (<0)",
			payload: func() RuntimeConfig {
				c := getValidRuntimeConfig()
				c.SensorLED.LedRGB = []float64{-10, 0, 0}
				return c
			}(),
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be between 0 and 255",
			shouldModify: false,
		},
		{
			name: "Negative Duration",
			payload: func() RuntimeConfig {
				c := getValidRuntimeConfig()
				c.SensorLED.RunUpDelay = -5 * time.Second
				return c
			}(),
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be non-negative",
			shouldModify: false,
		},
		{
			name: "Cylon Width Too Large",
			payload: func() RuntimeConfig {
				c := getValidRuntimeConfig()
				c.CylonLED.Enabled = true
				c.CylonLED.Width = 60 // LedsTotal is 100, max width is 50
				return c
			}(),
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "cannot be larger than half of LedsTotal",
			shouldModify: false,
		},
		{
			name: "BlobCfg X Out of Bounds",
			payload: func() RuntimeConfig {
				c := getValidRuntimeConfig()
				c.MultiBlobLED.Enabled = true
				c.MultiBlobLED.BlobCfg = []BlobCfg{
					{X: 100, Width: 10, LedRGB: []float64{0, 0, 0}}, // 100 is invalid index (0-99)
				}
				return c
			}(),
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be between 0 and 99",
			shouldModify: false,
		},
		{
			name: "Audio MinDB >= MaxDB",
			payload: func() RuntimeConfig {
				c := getValidRuntimeConfig()
				c.AudioLED.Enabled = true
				c.AudioLED.MinDB = -10
				c.AudioLED.MaxDB = -20
				return c
			}(),
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be less than MaxDB",
			shouldModify: false,
		},
	}

	// 3. Run Tests
	handler := ConfigHandler(configFile)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize payload to JSON
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/config", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Assert Status
			assert.Equal(t, tt.wantStatus, w.Code)

			// Assert Error Message
			if tt.wantErrorMsg != "" {
				assert.Contains(t, w.Body.String(), tt.wantErrorMsg)
			}

			// Assert File State
			currentConfig, err := ReadConfig(configFile)
			assert.NoError(t, err)

			if !tt.shouldModify {
				// Verify critical fields haven't changed to invalid values
				if strings.Contains(tt.name, "RGB") {
					assert.NotEqual(t, tt.payload.SensorLED.LedRGB, currentConfig.SensorLED.LedRGB, "File should not be updated with invalid RGB")
				}
			} else {
				// For valid update, check if it stuck
				if tt.payload.SensorLED.HoldTime > 0 {
					assert.Equal(t, tt.payload.SensorLED.HoldTime, currentConfig.SensorLED.HoldTime)
				}
			}
		})
	}
}