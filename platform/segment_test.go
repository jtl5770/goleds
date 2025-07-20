package platform

import (
	"reflect"
	"testing"

	"lautenbacher.net/goleds/producer"
)

func TestNewSegment(t *testing.T) {
	// Test basic segment creation
	s := NewSegment(0, 9, "spi1", false, true, 100)
	if s.FirstLed != 0 {
		t.Errorf("Expected FirstLed to be 0, got %d", s.FirstLed)
	}
	if s.LastLed != 9 {
		t.Errorf("Expected LastLed to be 9, got %d", s.LastLed)
	}
	if s.SpiMultiplex != "spi1" {
		t.Errorf("Expected SpiMultiplex to be 'spi1', got %s", s.SpiMultiplex)
	}
	if s.Reverse != false {
		t.Errorf("Expected Reverse to be false, got %t", s.Reverse)
	}
	if s.Visible != true {
		t.Errorf("Expected Visible to be true, got %t", s.Visible)
	}

	// Test with reversed led indices on creation
	s = NewSegment(9, 0, "spi1", false, true, 100)
	if s.FirstLed != 0 {
		t.Errorf("Expected FirstLed to be 0 after auto-reversal, got %d", s.FirstLed)
	}
	if s.LastLed != 9 {
		t.Errorf("Expected LastLed to be 9 after auto-reversal, got %d", s.LastLed)
	}

	// Test clamping for first and last led
	s = NewSegment(-5, 105, "spi1", false, true, 100)
	if s.FirstLed != 0 {
		t.Errorf("Expected FirstLed to be clamped to 0, got %d", s.FirstLed)
	}
	if s.LastLed != 99 {
		t.Errorf("Expected LastLed to be clamped to 99, got %d", s.LastLed)
	}

	// Test invisible segment
	s = NewSegment(0, 9, "spi1", false, false, 100)
	if s.SpiMultiplex != "__" {
		t.Errorf("Expected SpiMultiplex to be '__' for invisible segment, got %s", s.SpiMultiplex)
	}
}

func TestSetLeds(t *testing.T) {
	leds := make([]producer.Led, 10)
	for i := 0; i < 10; i++ {
		leds[i] = producer.Led{Red: float64(i)}
	}

	s := NewSegment(2, 5, "spi1", false, true, 10)
	s.SetLeds(leds)

	expected := []producer.Led{
		{Red: 2},
		{Red: 3},
		{Red: 4},
		{Red: 5},
	}

	if !reflect.DeepEqual(s.Leds, expected) {
		t.Errorf("Expected Leds to be %+v, got %+v", expected, s.Leds)
	}
}

func TestSetLedsReversed(t *testing.T) {
	leds := make([]producer.Led, 10)
	for i := 0; i < 10; i++ {
		leds[i] = producer.Led{Red: float64(i)}
	}

	s := NewSegment(2, 5, "spi1", true, true, 10)
	s.SetLeds(leds)

	expected := []producer.Led{
		{Red: 5},
		{Red: 4},
		{Red: 3},
		{Red: 2},
	}

	if !reflect.DeepEqual(s.Leds, expected) {
		t.Errorf("Expected Leds to be %+v, got %+v", expected, s.Leds)
	}
}

func TestGetLeds(t *testing.T) {
	s := NewSegment(0, 9, "spi1", false, true, 100)
	leds := make([]producer.Led, 10)
	s.Leds = leds
	if !reflect.DeepEqual(s.GetLeds(), leds) {
		t.Errorf("Expected GetLeds to return %+v, got %+v", leds, s.GetLeds())
	}

	s.Visible = false
	if s.GetLeds() != nil {
		t.Errorf("Expected GetLeds to return nil for invisible segment, got %+v", s.GetLeds())
	}
}