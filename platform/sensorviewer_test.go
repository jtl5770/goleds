package platform

import (
	"math"
	"testing"
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
