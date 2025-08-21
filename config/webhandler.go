package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

// configHandler routes API requests for /api/config to the appropriate handler
// based on the HTTP method. It also passes the config file path to the handlers.
func ConfigHandler(cfile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getConfigHandler(w, r, cfile)
		case http.MethodPost:
			setConfigHandler(w, r, cfile)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// getConfigHandler reads the current config file, extracts the runtime-safe
// configuration, and returns it as JSON.
func getConfigHandler(w http.ResponseWriter, r *http.Request, cfile string) {
	slog.Info("Handling GET /api/config request")
	// We read the file on every request to ensure we always have the latest version.
	// The realp and sensp flags are false because they don't affect reading the producer settings.
	fullConfig, err := ReadConfig(cfile, false, false)
	if err != nil {
		slog.Error("Failed to read config file for API", "error", err)
		http.Error(w, "Failed to read configuration", http.StatusInternalServerError)
		return
	}

	runtimeConfig := RuntimeConfig{
		SensorLED:    fullConfig.SensorLED,
		NightLED:     fullConfig.NightLED,
		ClockLED:     fullConfig.ClockLED,
		AudioLED:     fullConfig.AudioLED,
		CylonLED:     fullConfig.CylonLED,
		MultiBlobLED: fullConfig.MultiBlobLED,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(runtimeConfig); err != nil {
		slog.Error("Failed to encode runtime config to JSON", "error", err)
		http.Error(w, "Failed to serialize configuration", http.StatusInternalServerError)
	}
}

// setConfigHandler receives a JSON payload with runtime configuration, merges it
// with the full configuration on disk, validates it, and writes it back.
func setConfigHandler(w http.ResponseWriter, r *http.Request, cfile string) {
	slog.Info("Handling POST /api/config request")
	// 1. Decode the incoming JSON into our RuntimeConfig struct.
	var newRuntimeConfig RuntimeConfig
	if err := json.NewDecoder(r.Body).Decode(&newRuntimeConfig); err != nil {
		slog.Error("Failed to decode incoming JSON", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 2. Read the current full configuration from disk to preserve hardware settings.
	// The realp and sensp flags are false because we are just reading and merging.
	fullConfig, err := ReadConfig(cfile, false, false)
	if err != nil {
		slog.Error("Failed to read existing config for update", "error", err)
		http.Error(w, "Failed to read configuration", http.StatusInternalServerError)
		return
	}

	// 3. Merge the new runtime settings into the full config object.
	fullConfig.SensorLED = newRuntimeConfig.SensorLED
	fullConfig.NightLED = newRuntimeConfig.NightLED
	fullConfig.ClockLED = newRuntimeConfig.ClockLED
	fullConfig.AudioLED = newRuntimeConfig.AudioLED
	fullConfig.CylonLED = newRuntimeConfig.CylonLED
	fullConfig.MultiBlobLED = newRuntimeConfig.MultiBlobLED

	// 4. Validate the newly merged configuration.
	if err := fullConfig.Validate(); err != nil {
		slog.Error("Validation failed for new config", "error", err)
		http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	// 5. Marshal the full config back to YAML.
	yamlData, err := yaml.Marshal(&fullConfig)
	if err != nil {
		slog.Error("Failed to marshal merged config to YAML", "error", err)
		http.Error(w, "Failed to prepare configuration for saving", http.StatusInternalServerError)
		return
	}

	// 6. Write the new YAML back to the config file, triggering the reload.
	if err := os.WriteFile(cfile, yamlData, 0o644); err != nil {
		slog.Error("Failed to write updated config file", "error", err)
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	slog.Info("Successfully updated config file, application will reload.")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Configuration updated successfully.")
}
