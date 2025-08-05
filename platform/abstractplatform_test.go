package platform

import (
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
