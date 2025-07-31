package platform

import (
	"reflect"
	"testing"

	"lautenbacher.net/goleds/producer"
)

func TestNewSegment(t *testing.T) {
	// Test basic segment creation
	s := newSegment(0, 9, "spi1", false, true, 100)
	if s.firstLed != 0 {
		t.Errorf("Expected FirstLed to be 0, got %d", s.firstLed)
	}
	if s.lastLed != 9 {
		t.Errorf("Expected LastLed to be 9, got %d", s.lastLed)
	}
	if s.spiMultiplex != "spi1" {
		t.Errorf("Expected SpiMultiplex to be 'spi1', got %s", s.spiMultiplex)
	}
	if s.reverse != false {
		t.Errorf("Expected Reverse to be false, got %t", s.reverse)
	}
	if s.visible != true {
		t.Errorf("Expected Visible to be true, got %t", s.visible)
	}

	// Test with reversed led indices on creation
	s = newSegment(9, 0, "spi1", false, true, 100)
	if s.firstLed != 0 {
		t.Errorf("Expected FirstLed to be 0 after auto-reversal, got %d", s.firstLed)
	}
	if s.lastLed != 9 {
		t.Errorf("Expected LastLed to be 9 after auto-reversal, got %d", s.lastLed)
	}

	// Test clamping for first and last led
	s = newSegment(-5, 105, "spi1", false, true, 100)
	if s.firstLed != 0 {
		t.Errorf("Expected FirstLed to be clamped to 0, got %d", s.firstLed)
	}
	if s.lastLed != 99 {
		t.Errorf("Expected LastLed to be clamped to 99, got %d", s.lastLed)
	}

	// Test invisible segment
	s = newSegment(0, 9, "spi1", false, false, 100)
	if s.spiMultiplex != "__" {
		t.Errorf("Expected SpiMultiplex to be '__' for invisible segment, got %s", s.spiMultiplex)
	}
}

func TestSetLeds(t *testing.T) {
	leds := make([]producer.Led, 10)
	for i := 0; i < 10; i++ {
		leds[i] = producer.Led{Red: float64(i)}
	}

	s := newSegment(2, 5, "spi1", false, true, 10)
	s.setLeds(leds)

	expected := []producer.Led{
		{Red: 2},
		{Red: 3},
		{Red: 4},
		{Red: 5},
	}

	if !reflect.DeepEqual(s.leds, expected) {
		t.Errorf("Expected Leds to be %+v, got %+v", expected, s.leds)
	}
}

func TestSetLedsReversed(t *testing.T) {
	leds := make([]producer.Led, 10)
	for i := 0; i < 10; i++ {
		leds[i] = producer.Led{Red: float64(i)}
	}

	s := newSegment(2, 5, "spi1", true, true, 10)
	s.setLeds(leds)

	expected := []producer.Led{
		{Red: 5},
		{Red: 4},
		{Red: 3},
		{Red: 2},
	}

	if !reflect.DeepEqual(s.leds, expected) {
		t.Errorf("Expected Leds to be %+v, got %+v", expected, s.leds)
	}
}

func TestGetLeds(t *testing.T) {
	s := newSegment(0, 9, "spi1", false, true, 100)
	leds := make([]producer.Led, 10)
	s.leds = leds
	if !reflect.DeepEqual(s.getLeds(), leds) {
		t.Errorf("Expected GetLeds to return %+v, got %+v", leds, s.getLeds())
	}

	s.visible = false
	if s.getLeds() != nil {
		t.Errorf("Expected GetLeds to return nil for invisible segment, got %+v", s.getLeds())
	}
}
