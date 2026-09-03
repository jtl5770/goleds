package platform

import (
	"os"
	"reflect"
	"testing"
	"time"

	"lautenbacher.net/goleds/config"
	"lautenbacher.net/goleds/producer"
)

func TestNewColorLUT(t *testing.T) {
	lut := newColorLUT([]float64{1.0, 0.5, 2.0})

	if lut.r[100] != 100 {
		t.Errorf("Expected r[100]=100, got %d", lut.r[100])
	}
	if lut.g[100] != 50 {
		t.Errorf("Expected g[100]=50, got %d", lut.g[100])
	}
	// 100 * 2.0 = 200; 200 * 2.0 = 400 clamped to 255
	if lut.b[100] != 200 {
		t.Errorf("Expected b[100]=200, got %d", lut.b[100])
	}
	if lut.b[200] != 255 {
		t.Errorf("Expected b[200] clamped to 255, got %d", lut.b[200])
	}

	// Empty correction defaults to 1.0, 1.0, 1.0
	lutDefault := newColorLUT(nil)
	if lutDefault.r[123] != 123 || lutDefault.g[123] != 123 || lutDefault.b[123] != 123 {
		t.Errorf("Default LUT mismatch")
	}
}

func TestRaspberryPiPlatform_Init(t *testing.T) {
	cfg := &config.Config{
		Hardware: config.HardwareConfig{
			Display: config.DisplayConfig{
				LedsTotal:        50,
				ForceUpdateDelay: 100 * time.Millisecond,
			},
		},
	}

	rpi := NewRaspberryPiPlatform(cfg)
	if rpi == nil {
		t.Fatal("Expected NewRaspberryPiPlatform to return non-nil")
	}

	sigChan := make(chan os.Signal, 1)
	sv := NewSensorViewer(cfg.Hardware.Sensors, sigChan, false)
	rpi.SetSensorViewer(sv)
	if rpi.sensorViewer != sv {
		t.Error("SetSensorViewer did not set sensor viewer")
	}
}

func TestWS2801Driver_Write(t *testing.T) {
	displayConfig := config.DisplayConfig{
		ColorCorrection: []float64{1.0, 1.0, 1.0},
		LedsTotal:       10, // Set a realistic total for buffer allocation
	}
	driver := newWs2801Driver(displayConfig)

	segment := &segment{
		leds: []producer.Led{
			{Red: 255, Green: 0, Blue: 0},
			{Red: 0, Green: 255, Blue: 0},
			{Red: 0, Green: 0, Blue: 255},
		},
		spiMultiplex: "spi1",
	}

	var sentData []byte
	exchangeFunc := func(index string, data []byte) []byte {
		sentData = data
		return data
	}

	err := driver.write(segment, exchangeFunc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	expected := []byte{255, 0, 0, 0, 255, 0, 0, 0, 255}
	if !reflect.DeepEqual(sentData, expected) {
		t.Errorf("Expected data %v, got %v", expected, sentData)
	}
}

func TestWS2801Driver_Write_Reversed(t *testing.T) {
	displayConfig := config.DisplayConfig{
		ColorCorrection: []float64{1.0, 1.0, 1.0},
		LedsTotal:       10,
	}
	driver := newWs2801Driver(displayConfig)

	segment := &segment{
		leds: []producer.Led{
			{Red: 255, Green: 0, Blue: 0},
			{Red: 0, Green: 255, Blue: 0},
			{Red: 0, Green: 0, Blue: 255},
		},
		reverse:      true,
		spiMultiplex: "spi1",
	}

	var sentData []byte
	exchangeFunc := func(index string, data []byte) []byte {
		sentData = data
		return data
	}

	err := driver.write(segment, exchangeFunc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// In reverse mode, the 3rd LED (Blue) is written first, then Green, then Red.
	expected := []byte{0, 0, 255, 0, 255, 0, 255, 0, 0}
	if !reflect.DeepEqual(sentData, expected) {
		t.Errorf("Expected data %v, got %v", expected, sentData)
	}
}

func TestAPA102Driver_Write(t *testing.T) {
	displayConfig := config.DisplayConfig{
		ColorCorrection:   []float64{1.0, 1.0, 1.0},
		APA102_Brightness: 31,
		LedsTotal:         10, // Set a realistic total for buffer allocation
	}
	driver := newApa102Driver(displayConfig)

	segment := &segment{
		leds: []producer.Led{
			{Red: 255, Green: 0, Blue: 0},
			{Red: 0, Green: 255, Blue: 0},
		},
		spiMultiplex: "spi1",
	}

	var sentData []byte
	exchangeFunc := func(index string, data []byte) []byte {
		sentData = data
		return data
	}

	err := driver.write(segment, exchangeFunc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Expected:
	// 4 bytes start frame (0x00, 0x00, 0x00, 0x00)
	// For each LED:
	//   1 byte brightness (0xE0 | 31 = 0xFF)
	//   3 bytes color (blue, green, red)
	// frame end: at least (len(values) / 2) + 1 bits of 0xFF
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

func TestAPA102Driver_Write_Reversed(t *testing.T) {
	displayConfig := config.DisplayConfig{
		ColorCorrection:   []float64{1.0, 1.0, 1.0},
		APA102_Brightness: 31,
		LedsTotal:         10,
	}
	driver := newApa102Driver(displayConfig)

	segment := &segment{
		leds: []producer.Led{
			{Red: 255, Green: 0, Blue: 0},
			{Red: 0, Green: 255, Blue: 0},
		},
		reverse:      true,
		spiMultiplex: "spi1",
	}

	var sentData []byte
	exchangeFunc := func(index string, data []byte) []byte {
		sentData = data
		return data
	}

	err := driver.write(segment, exchangeFunc)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// In reverse mode, LED 2 (Green) is written at index 0, LED 1 (Red) at index 1.
	expected := []byte{
		0x00, 0x00, 0x00, 0x00, // Start frame
		0xFF, 0, 255, 0, // LED 2 (Green)
		0xFF, 0, 0, 255, // LED 1 (Red)
		0xFF, // End frame
	}

	if !reflect.DeepEqual(sentData, expected) {
		t.Errorf("Expected data %v, got %v", expected, sentData)
	}
}

func BenchmarkWS2801Driver_Write(b *testing.B) {
	displayConfig := config.DisplayConfig{
		ColorCorrection: []float64{1.0, 0.9, 0.8},
		LedsTotal:       300,
	}
	driver := newWs2801Driver(displayConfig)
	seg := &segment{
		leds:         make([]producer.Led, 300),
		reverse:      true,
		spiMultiplex: "spi1",
	}
	for i := range seg.leds {
		seg.leds[i] = producer.Led{Red: float64(i % 255), Green: float64((i * 2) % 255), Blue: 100}
	}
	noopExchange := func(index string, data []byte) []byte { return data }

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = driver.write(seg, noopExchange)
	}
}

func BenchmarkAPA102Driver_Write(b *testing.B) {
	displayConfig := config.DisplayConfig{
		ColorCorrection:   []float64{1.0, 0.9, 0.8},
		APA102_Brightness: 31,
		LedsTotal:         300,
	}
	driver := newApa102Driver(displayConfig)
	seg := &segment{
		leds:         make([]producer.Led, 300),
		reverse:      true,
		spiMultiplex: "spi1",
	}
	for i := range seg.leds {
		seg.leds[i] = producer.Led{Red: float64(i % 255), Green: float64((i * 2) % 255), Blue: 100}
	}
	noopExchange := func(index string, data []byte) []byte { return data }

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = driver.write(seg, noopExchange)
	}
}
