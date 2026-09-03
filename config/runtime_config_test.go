package config

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestRuntimeConfig_YAMLAndJSON(t *testing.T) {
	rc := RuntimeConfig{
		LedsTotal: 100,
		SensorLED: SensorLEDConfig{
			Enabled:    true,
			RunUpDelay: 10 * time.Millisecond,
			LedRGB:     []float64{100, 200, 50},
		},
		NightLED: NightLEDConfig{
			Enabled:   false,
			Latitude:  48.137,
			Longitude: 11.576,
		},
		AudioLED: AudioLEDConfig{
			Enabled: true,
			MinDB:   -60,
			MaxDB:   -6,
		},
	}

	// Test JSON marshal / unmarshal
	jsonData, err := json.Marshal(rc)
	if err != nil {
		t.Fatalf("Failed to marshal RuntimeConfig to JSON: %v", err)
	}

	var jsonRC RuntimeConfig
	if err := json.Unmarshal(jsonData, &jsonRC); err != nil {
		t.Fatalf("Failed to unmarshal RuntimeConfig from JSON: %v", err)
	}

	if jsonRC.LedsTotal != 100 || !jsonRC.SensorLED.Enabled || jsonRC.AudioLED.MinDB != -60 {
		t.Errorf("Mismatch in unmarshaled JSON RuntimeConfig: %+v", jsonRC)
	}

	// Test YAML marshal / unmarshal
	yamlData, err := yaml.Marshal(rc)
	if err != nil {
		t.Fatalf("Failed to marshal RuntimeConfig to YAML: %v", err)
	}

	var yamlRC RuntimeConfig
	if err := yaml.Unmarshal(yamlData, &yamlRC); err != nil {
		t.Fatalf("Failed to unmarshal RuntimeConfig from YAML: %v", err)
	}

	if yamlRC.LedsTotal != 100 || !yamlRC.AudioLED.Enabled {
		t.Errorf("Mismatch in unmarshaled YAML RuntimeConfig: %+v", yamlRC)
	}
}
