package platform

import (
	"math"
	"testing"
)

func TestSensor_smoothValue(t *testing.T) {
	s := &sensor{
		capacity: 5,
		values:   make([]int, 5),
	}

	// Initial values are all 0
	if avg := s.smoothedValue(10); avg != 2 {
		t.Errorf("Expected smoothed value to be 2, got %d", avg)
	}
	// values: [0, 0, 0, 0, 10] -> sum=10, avg=2

	if avg := s.smoothedValue(20); avg != 6 {
		t.Errorf("Expected smoothed value to be 6, got %d", avg)
	}
	// values: [0, 0, 0, 10, 20] -> sum=30, avg=6

	if avg := s.smoothedValue(30); avg != 12 {
		t.Errorf("Expected smoothed value to be 12, got %d", avg)
	}
	// values: [0, 0, 10, 20, 30] -> sum=60, avg=12

	if avg := s.smoothedValue(40); avg != 20 {
		t.Errorf("Expected smoothed value to be 20, got %d", avg)
	}
	// values: [0, 10, 20, 30, 40] -> sum=100, avg=20

	if avg := s.smoothedValue(50); avg != 30 {
		t.Errorf("Expected smoothed value to be 30, got %d", avg)
	}
	// values: [10, 20, 30, 40, 50] -> sum=150, avg=30

	if avg := s.smoothedValue(0); avg != 28 {
		t.Errorf("Expected smoothed value to be 28, got %d", avg)
	}
	// values: [20, 30, 40, 50, 0] -> sum=140, avg=28
}

func TestSensor_thresholdForRed(t *testing.T) {
	s := &sensor{}

	// Uncalibrated sensor returns math.MaxInt
	if th := s.thresholdForRed(70); th != math.MaxInt {
		t.Errorf("Expected math.MaxInt for uncalibrated sensor, got %d", th)
	}

	s.setCalibrationCurve([]CalibPoint{
		{Red: 70, Threshold: 500},
		{Red: 47, Threshold: 400},
		{Red: 23, Threshold: 300},
		{Red: 0, Threshold: 200},
	})

	// Above max calibrated red
	if th := s.thresholdForRed(100); th != 500 {
		t.Errorf("Expected 500 for red 100, got %d", th)
	}
	// Exact points
	if th := s.thresholdForRed(70); th != 500 {
		t.Errorf("Expected 500 for red 70, got %d", th)
	}
	if th := s.thresholdForRed(47); th != 400 {
		t.Errorf("Expected 400 for red 47, got %d", th)
	}
	if th := s.thresholdForRed(23); th != 300 {
		t.Errorf("Expected 300 for red 23, got %d", th)
	}
	if th := s.thresholdForRed(0); th != 200 {
		t.Errorf("Expected 200 for red 0, got %d", th)
	}
	// Interpolated point between 23 (300) and 0 (200) -> at ~11 should be ~248
	th11 := s.thresholdForRed(11)
	if th11 < 240 || th11 > 260 {
		t.Errorf("Expected ~248 for red 11, got %d", th11)
	}
}
