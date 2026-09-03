package platform

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
)

func TestCalculateStats(t *testing.T) {
	data := []int{10, 20, 30, 40, 50}

	stats := calculateStats(data)

	// Expected values
	expectedMin := 10
	expectedMax := 50
	expectedMean := 30.0
	expectedMedian := 30.0
	expectedStdDev := math.Sqrt(200) // sqrt(((10-30)^2 + (20-30)^2 + (30-30)^2 + (40-30)^2 + (50-30)^2) / 5) = sqrt( (400+100+0+100+400)/5 ) = sqrt(1000/5) = sqrt(200)

	if stats.min != expectedMin {
		t.Errorf("Expected min to be %d, got %d", expectedMin, stats.min)
	}
	if stats.max != expectedMax {
		t.Errorf("Expected max to be %d, got %d", expectedMax, stats.max)
	}
	if stats.mean != expectedMean {
		t.Errorf("Expected mean to be %.2f, got %.2f", expectedMean, stats.mean)
	}
	if stats.median != expectedMedian {
		t.Errorf("Expected median to be %.2f, got %.2f", expectedMedian, stats.median)
	}
	if math.Abs(stats.stdDev-expectedStdDev) > 1e-9 {
		t.Errorf("Expected stdDev to be %.2f, got %.2f", expectedStdDev, stats.stdDev)
	}
}

func TestCalculateStats_Empty(t *testing.T) {
	data := []int{}
	stats := calculateStats(data)
	if stats.min != 0 || stats.max != 0 || stats.mean != 0 || stats.median != 0 || stats.stdDev != 0 {
		t.Errorf("Expected all stats to be 0 for empty data, got %+v", stats)
	}
}

func TestCalculateStats_EvenLength(t *testing.T) {
	data := []int{10, 20, 30, 40}
	stats := calculateStats(data)
	expectedMedian := 25.0
	if stats.median != expectedMedian {
		t.Errorf("Expected median for even length data to be %.2f, got %.2f", expectedMedian, stats.median)
	}
}

func TestSensorViewer_NewAndUpdate(t *testing.T) {
	cfg := config.SensorsConfig{
		LoopDelay: 10 * time.Millisecond,
		SensorCfg: map[string]config.SensorCfg{
			"S2": {LedIndex: 20},
			"S1": {LedIndex: 10},
		},
	}

	sigChan := make(chan os.Signal, 1)
	sv := NewSensorViewer(cfg, sigChan, false)

	// Verify sensor names are sorted by LedIndex (S1 before S2)
	if len(sv.sensorNames) != 2 || sv.sensorNames[0] != "S1" || sv.sensorNames[1] != "S2" {
		t.Errorf("Expected sorted sensor names [S1, S2], got %v", sv.sensorNames)
	}

	// Update with history
	sv.mu.Lock()
	for i := 1; i <= 550; i++ {
		q := sv.sensorValues["S1"]
		if q.Len() == maxSensorHistory {
			q.PopFront()
		}
		q.PushBack(i)
	}
	if sv.sensorValues["S1"].Len() != maxSensorHistory {
		t.Errorf("Expected queue capped at %d, got %d", maxSensorHistory, sv.sensorValues["S1"].Len())
	}

	line1, line2, line3 := sv.prepareDisplayStrings()
	sv.mu.Unlock()

	if !strings.Contains(line1, "min|mean|max") {
		t.Errorf("Expected line1 to contain 'min|mean|max', got: %s", line1)
	}
	if !strings.Contains(line2, "Standard Deviation") {
		t.Errorf("Expected line2 to contain 'Standard Deviation', got: %s", line2)
	}
	if !strings.Contains(line3, "S1") || !strings.Contains(line3, "S2") {
		t.Errorf("Expected line3 to contain sensor names, got: %s", line3)
	}

	sv.Stop()
}
