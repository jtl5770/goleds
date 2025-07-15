package rpi

import (
	"reflect"
	"testing"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/platform"
	"lautenbacher.net/goleds/producer"
)

func TestWS2801Driver_Write(t *testing.T) {
	displayConfig := config.DisplayConfig{
		ColorCorrection: []float64{1.0, 1.0, 1.0},
	}
	driver := newWS2801Driver(displayConfig)

	segment := &platform.Segment{
		Leds: []producer.Led{
			{Red: 255, Green: 0, Blue: 0},
			{Red: 0, Green: 255, Blue: 0},
			{Red: 0, Green: 0, Blue: 255},
		},
		SpiMultiplex: "spi1",
	}

	var sentData []byte
	exchangeFunc := func(index string, data []byte) []byte {
		sentData = data
		return data
	}

	err := driver.Write(segment, exchangeFunc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	expected := []byte{255, 0, 0, 0, 255, 0, 0, 0, 255}
	if !reflect.DeepEqual(sentData, expected) {
		t.Errorf("Expected data %v, got %v", expected, sentData)
	}
}

func TestAPA102Driver_Write(t *testing.T) {
	displayConfig := config.DisplayConfig{
		ColorCorrection:   []float64{1.0, 1.0, 1.0},
		APA102_Brightness: 31,
	}
	driver := newAPA102Driver(displayConfig)

	segment := &platform.Segment{
		Leds: []producer.Led{
			{Red: 255, Green: 0, Blue: 0},
			{Red: 0, Green: 255, Blue: 0},
		},
		SpiMultiplex: "spi1",
	}

	var sentData []byte
	exchangeFunc := func(index string, data []byte) []byte {
		sentData = data
		return data
	}

	err := driver.Write(segment, exchangeFunc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Expected:
	// 4 bytes start frame (0x00, 0x00, 0x00, 0x00)
	// For each LED:
	//   1 byte brightness (0xE0 | 31 = 0xFF)
	//   3 bytes color (blue, green, red)
	// 1 byte end frame (0xFF)
	expected := []byte{
		0x00, 0x00, 0x00, 0x00, // Start frame
		0xFF, 0, 0, 255, // LED 1
		0xFF, 0, 255, 0, // LED 2
		0xFF, // End frame
	}

	if !reflect.DeepEqual(sentData, expected) {
		t.Errorf("Expected data %v, got %v", expected, sentData)
	}
}
