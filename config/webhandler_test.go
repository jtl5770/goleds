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

func TestConfigHandler_SetValidation(t *testing.T) {
	// 1. Setup temporary environment
	tempDir, err := os.MkdirTemp("", "goleds-webtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.yml")
	
	// Create a valid initial configuration
	initialConfig := Config{
		Hardware: HardwareConfig{
			Display: DisplayConfig{LedsTotal: 100},
		},
		SensorLED: SensorLEDConfig{
			Enabled: true,
			LedRGB: []float64{0,0,0},
			HoldTime: 5 * time.Second,
		},
	}
	// We need to write this as proper YAML first
	data, _ := yaml.Marshal(initialConfig)
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// 2. Define Test Cases
	tests := []struct {
		name           string
		payload        RuntimeConfig
		wantStatus     int
		wantErrorMsg   string
		shouldModify   bool
	}{
		{
			name: "Valid Update",
			payload: RuntimeConfig{
				SensorLED: SensorLEDConfig{
					Enabled: true,
					LedRGB: []float64{50, 50, 50},
					HoldTime: 10 * time.Second,
				},
			},
			wantStatus:   http.StatusOK,
			shouldModify: true,
		},
		{
			name: "Invalid RGB (>255)",
			payload: RuntimeConfig{
				SensorLED: SensorLEDConfig{
					Enabled: true,
					LedRGB: []float64{300, 0, 0}, 
				},
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be between 0 and 255",
			shouldModify: false,
		},
		{
			name: "Invalid RGB (<0)",
			payload: RuntimeConfig{
				SensorLED: SensorLEDConfig{
					Enabled: true,
					LedRGB: []float64{-10, 0, 0},
				},
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be between 0 and 255",
			shouldModify: false,
		},
		{
			name: "Negative Duration",
			payload: RuntimeConfig{
				SensorLED: SensorLEDConfig{
					Enabled: true,
					LedRGB: []float64{0,0,0},
					RunUpDelay: -5 * time.Second,
				},
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be non-negative",
			shouldModify: false,
		},
		{
			name: "Cylon Width Too Large",
			payload: RuntimeConfig{
				SensorLED: SensorLEDConfig{Enabled: true, LedRGB: []float64{0,0,0}}, // Keep base valid
				CylonLED: CylonLEDConfig{
					Enabled: true,
					Width: 60, // LedsTotal is 100, max width is 50
					Step: 1,
					LedRGB: []float64{0,0,0},
				},
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "cannot be larger than half of LedsTotal",
			shouldModify: false,
		},
		{
			name: "BlobCfg X Out of Bounds",
			payload: RuntimeConfig{
				SensorLED: SensorLEDConfig{Enabled: true, LedRGB: []float64{0,0,0}},
				MultiBlobLED: MultiBlobLEDConfig{
					Enabled: true,
					BlobCfg: []BlobCfg{
						{X: 100, Width: 10, LedRGB: []float64{0,0,0}}, // 100 is invalid index (0-99)
					},
				},
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be between 0 and 99",
			shouldModify: false,
		},
		{
			name: "Audio MinDB >= MaxDB",
			payload: RuntimeConfig{
				SensorLED: SensorLEDConfig{Enabled: true, LedRGB: []float64{0,0,0}},
				AudioLED: AudioLEDConfig{
					Enabled: true,
					SampleRate: 44100, FramesPerBuffer: 1024,
					MinDB: -10,
					MaxDB: -20, // Invalid: Max < Min
					LedGreen: []float64{0,0,0}, LedYellow: []float64{0,0,0}, LedRed: []float64{0,0,0},
				},
			},
			wantStatus:   http.StatusBadRequest,
			wantErrorMsg: "must be less than MaxDB",
			shouldModify: false,
		},
	}

	// 3. Run Tests
	handler := ConfigHandler(configFile)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config file to known state before each test if necessary, 
			// but for this sequence, we rely on failures NOT changing the state.
			// Only "Valid Update" changes it, which is fine as subsequent tests overwrite fields they care about.
			
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
				// If we expected failure, the SensorLED.RunUpDelay should NOT match the payload if the payload was invalid
				// We can check specific fields based on the test case, but a general check:
				// If we tried to set RunUpDelay to -5s, reading it back should NOT be -5s (it physically can't be stored often, but logic holds)
				// More concretely: The RGB check.
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
