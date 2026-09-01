package config

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// createDummyConfigFile creates a temporary config file with valid dummy data for testing.
func createDummyConfigFile(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	cfile := filepath.Join(tempDir, "test_config.yml")

	// Minimal valid configuration
	conf := Config{
		SensorLED: SensorLEDConfig{
			Enabled:           true,
			RunUpDelay:        10 * time.Millisecond,
			RunDownDelay:      10 * time.Millisecond,
			HoldTime:          10 * time.Millisecond,
			LedRGB:            []float64{100, 100, 100},
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
			Enabled:       false,
			StartLedLeft:  0,
			EndLedLeft:    1,
			StartLedRight: 2,
			EndLedRight:   3,
			LedGreen:      []float64{0, 0, 0},
			LedYellow:     []float64{0, 0, 0},
			LedRed:        []float64{0, 0, 0},
			UpdateFreq:    10 * time.Millisecond,
			MinDB:         -60,
			MaxDB:         -10,
			Squeezebox: SqueezeboxConfig{
				Server:        "127.0.0.1",
				SlimProtoPort: 3483,
				JSONRPCPort:   9000,
				PlayerMAC:     "00:04:20:11:22:33",
				PlayerName:    "Test VU",
				AutoSync:      true,
			},
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
			BlobCfg: []BlobCfg{
				{
					DeltaX: 1,
					X:      0,
					Width:  1,
					LedRGB: []float64{0, 0, 0},
				},
			},
		},
		Hardware: HardwareConfig{
			WebserverPort: 8080,
			LEDType:       "ws2801",
			SPIFrequency:  1000000,
			Display: DisplayConfig{
				ForceUpdateDelay:  1000 * time.Millisecond,
				LedsTotal:         10,
				ColorCorrection:   []float64{1, 1, 1},
				APA102_Brightness: 31,
				LedSegments: map[string][]LedSegmentConfig{
					"GroupA": {
						{
							FirstLed:     0,
							LastLed:      9,
							SpiMultiplex: "L1",
							Reverse:      false,
						},
					},
				},
			},
			Sensors: SensorsConfig{
				SmoothingSize: 5,
				LoopDelay:     50 * time.Millisecond,
				Calibration: CalibrationConfig{
					StepDuration:     10 * time.Second,
					MinMargin:        60,
					DeviationFactor:  1.75,
					OutlierThreshold: 150,
					RetryDelay:       5 * time.Second,
				},
				SensorCfg: map[string]SensorCfg{
					"S0": {
						LedIndex:     0,
						SpiMultiplex: "ADC1",
						AdcChannel:   0,
					},
				},
			},
			SpiMultiplexGPIO: map[string]struct {
				Low  []int `yaml:"Low,flow"`
				High []int `yaml:"High,flow"`
				CS   int   `yaml:"CS,flow"`
			}{
				"L1": {
					Low:  []int{17},
					High: []int{22, 23, 24},
				},
				"ADC1": {
					Low:  []int{17, 22, 23},
					High: []int{24},
				},
			},
		},
		Logging: LoggingConfig{
			TUI: SingleLoggingConfig{
				Level:  "INFO",
				Format: "text",
			},
			HW: SingleLoggingConfig{
				Level:  "INFO",
				Format: "text",
			},
		},
	}

	data, err := yaml.Marshal(&conf)
	if err != nil {
		t.Fatalf("Failed to marshal dummy config: %v", err)
	}

	if err := os.WriteFile(cfile, data, 0o644); err != nil {
		t.Fatalf("Failed to write dummy config file: %v", err)
	}

	return cfile
}

func TestConfigHandler_Get(t *testing.T) {
	cfile := createDummyConfigFile(t)
	handler := ConfigHandler(cfile)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK (200), got %d", resp.StatusCode)
	}

	var runtimeConf RuntimeConfig
	if err := json.NewDecoder(resp.Body).Decode(&runtimeConf); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if runtimeConf.LedsTotal != 10 {
		t.Errorf("Expected LedsTotal to be 10, got %d", runtimeConf.LedsTotal)
	}
	if !runtimeConf.SensorLED.Enabled {
		t.Errorf("Expected SensorLED.Enabled to be true, got %v", runtimeConf.SensorLED.Enabled)
	}
}

func TestConfigHandler_Post_Success(t *testing.T) {
	cfile := createDummyConfigFile(t)
	handler := ConfigHandler(cfile)

	// Create an updated RuntimeConfig payload
	updatedRuntimeConf := RuntimeConfig{
		LedsTotal: 10,
		SensorLED: SensorLEDConfig{
			Enabled:      true,
			RunUpDelay:   20 * time.Millisecond, // Changed
			RunDownDelay: 10 * time.Millisecond,
			HoldTime:     10 * time.Millisecond,
			LedRGB:       []float64{100, 100, 100},
			LatchLedRGB:  []float64{0, 0, 0},
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
			Enabled:       false,
			StartLedLeft:  0,
			EndLedLeft:    1,
			StartLedRight: 2,
			EndLedRight:   3,
			LedGreen:      []float64{0, 0, 0},
			LedYellow:     []float64{0, 0, 0},
			LedRed:        []float64{0, 0, 0},
			UpdateFreq:    10 * time.Millisecond,
			MinDB:         -60,
			MaxDB:         -10,
			Squeezebox: SqueezeboxConfig{
				Server:        "127.0.0.1",
				SlimProtoPort: 3483,
				JSONRPCPort:   9000,
				PlayerMAC:     "00:04:20:11:22:33",
				PlayerName:    "Test VU",
				AutoSync:      true,
			},
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
			BlobCfg: []BlobCfg{
				{
					DeltaX: 1,
					X:      0,
					Width:  1,
					LedRGB: []float64{0, 0, 0},
				},
			},
		},
	}

	payload, err := json.Marshal(updatedRuntimeConf)
	if err != nil {
		t.Fatalf("Failed to marshal updated config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK (200), got %d", resp.StatusCode)
	}

	// Read the file back and verify the changes were saved
	conf, err := ReadConfig(cfile)
	if err != nil {
		t.Fatalf("Failed to read updated config file: %v", err)
	}

	if conf.SensorLED.RunUpDelay != 20*time.Millisecond {
		t.Errorf("Expected SensorLED.RunUpDelay to be 20ms, got %v", conf.SensorLED.RunUpDelay)
	}
}

func TestConfigHandler_Post_InvalidData(t *testing.T) {
	cfile := createDummyConfigFile(t)
	handler := ConfigHandler(cfile)

	// Send an invalid JSON payload (RunUpDelay is negative, which fails validation)
	invalidRuntimeConf := RuntimeConfig{
		LedsTotal: 10,
		SensorLED: SensorLEDConfig{
			Enabled:     true,
			RunUpDelay:  -10 * time.Millisecond, // Invalid!
			LedRGB:      []float64{100, 100, 100},
			LatchLedRGB: []float64{0, 0, 0},
		},
	}

	payload, err := json.Marshal(invalidRuntimeConf)
	if err != nil {
		t.Fatalf("Failed to marshal invalid config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest (400) for invalid data, got %d", resp.StatusCode)
	}
}

func TestConfigHandler_MethodNotAllowed(t *testing.T) {
	cfile := createDummyConfigFile(t)
	handler := ConfigHandler(cfile)

	req := httptest.NewRequest(http.MethodDelete, "/api/config", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed (405), got %d", resp.StatusCode)
	}
}
