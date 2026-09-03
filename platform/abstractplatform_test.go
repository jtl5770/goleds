package platform

import (
	"math"
	"sync"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/producer"
)

func TestSensor_smoothValue(t *testing.T) {
	s := &sensor{
		values: make([]int, 5),
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

func TestAbstractPlatform_Methods(t *testing.T) {
	cfg := &config.Config{
		Hardware: config.HardwareConfig{
			Display: config.DisplayConfig{
				LedsTotal:        100,
				ForceUpdateDelay: 50 * time.Millisecond,
			},
		},
	}

	var mu sync.Mutex
	var displayedCount int
	displayFunc := func(leds []producer.Led) {
		mu.Lock()
		displayedCount = len(leds)
		mu.Unlock()
	}

	ap := newAbstractPlatform(cfg, displayFunc)
	ap.ledBufferPool = &sync.Pool{
		New: func() any {
			return make([]producer.Led, 10)
		},
	}
	ap.sensors["sensor1"] = &sensor{LedIndex: 15}

	if ap.GetLedsTotal() != 100 {
		t.Errorf("Expected LedsTotal 100, got %d", ap.GetLedsTotal())
	}
	if ap.GetForceUpdateDelay() != 50*time.Millisecond {
		t.Errorf("Expected ForceUpdateDelay 50ms, got %v", ap.GetForceUpdateDelay())
	}

	indices := ap.GetSensorLedIndices()
	if indices["sensor1"] != 15 {
		t.Errorf("Expected sensor1 index 15, got %d", indices["sensor1"])
	}

	if ap.IsCalibrating() {
		t.Error("Expected IsCalibrating to be false")
	}

	// Test displayDriver loop
	ap.displayWg.Add(1)
	go ap.displayDriver()

	testLeds := make([]producer.Led, 10)
	testLeds[0] = producer.Led{Red: 50, Green: 20, Blue: 10}
	ap.SetLeds(testLeds)

	time.Sleep(20 * time.Millisecond)
	if ap.getCurrentMaxRed() != 50 {
		t.Errorf("Expected currentMaxRed to be 50, got %d", ap.getCurrentMaxRed())
	}

	mu.Lock()
	count := displayedCount
	mu.Unlock()
	if count != 10 {
		t.Errorf("Expected displayedLeds len 10, got %d", count)
	}

	ap.displayStopChan <- true
	ap.displayWg.Wait()
}

func BenchmarkSensor_SmoothedValue(b *testing.B) {
	s := &sensor{
		values: make([]int, 16),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = s.smoothedValue(i & 0x3FF)
	}
}
