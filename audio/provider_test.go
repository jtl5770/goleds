package audio

import (
	"math"
	"sync"
	"testing"
)

func TestAtomicLevels_InitialState(t *testing.T) {
	al := NewAtomicLevels()
	left, right, active := al.Get()

	if left != -100 || right != -100 {
		t.Errorf("Expected initial levels (-100, -100), got (%.2f, %.2f)", left, right)
	}
	if active {
		t.Errorf("Expected initial active state to be false, got true")
	}
}

func TestAtomicLevels_SetAndGet(t *testing.T) {
	al := NewAtomicLevels()

	al.Set(-12.5, -6.0, true)
	left, right, active := al.Get()

	if math.Abs(left-(-12.5)) > 1e-9 {
		t.Errorf("Expected left -12.5, got %f", left)
	}
	if math.Abs(right-(-6.0)) > 1e-9 {
		t.Errorf("Expected right -6.0, got %f", right)
	}
	if !active {
		t.Errorf("Expected active true, got false")
	}

	al.Set(0.0, -3.0, false)
	left, right, active = al.Get()

	if left != 0.0 || right != -3.0 {
		t.Errorf("Expected levels (0.0, -3.0), got (%f, %f)", left, right)
	}
	if active {
		t.Errorf("Expected active false, got true")
	}
}

func TestAtomicLevels_ConcurrentAccess(t *testing.T) {
	al := NewAtomicLevels()
	var wg sync.WaitGroup

	iterations := 10000

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			val := float64(i % 100)
			al.Set(-val, -(val / 2), (i%2 == 0))
		}
	}()

	// Reader goroutines
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				left, right, _ := al.Get()
				if left > 0 || right > 0 {
					t.Errorf("Unexpected positive level: Left=%f, Right=%f", left, right)
				}
			}
		}()
	}

	wg.Wait()
}
