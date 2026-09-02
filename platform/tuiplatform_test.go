package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"lautenbacher.net/goleds/producer"
)

func TestScaledColor(t *testing.T) {
	// Empty/black
	assert.Equal(t, "[#000000]", scaledColor(producer.Led{Red: 0, Green: 0, Blue: 0}))

	// Pure Red scaled to max
	assert.Equal(t, "[#ff0000]", scaledColor(producer.Led{Red: 100, Green: 0, Blue: 0}))

	// Pure Green scaled to max
	assert.Equal(t, "[#00ff00]", scaledColor(producer.Led{Red: 0, Green: 50, Blue: 0}))

	// Pure Blue scaled to max
	assert.Equal(t, "[#0000ff]", scaledColor(producer.Led{Red: 0, Green: 0, Blue: 200}))

	// Mixed color (100, 50, 0) -> scales to (255, 128, 0) -> [#ff8000]
	assert.Equal(t, "[#ff8000]", scaledColor(producer.Led{Red: 100, Green: 50, Blue: 0}))
}

func TestRuneLUTs(t *testing.T) {
	// Threshold 0-3
	assert.Equal(t, " ", topRuneLUT[0])
	assert.Equal(t, "\u2581", bottomRuneLUT[0])

	// Threshold 24
	assert.Equal(t, " ", topRuneLUT[24])
	assert.Equal(t, "█", bottomRuneLUT[24])

	// Threshold 27
	assert.Equal(t, "\u2581", topRuneLUT[27])
	assert.Equal(t, "█", bottomRuneLUT[27])

	// Max value 255
	assert.Equal(t, "▒", topRuneLUT[255])
	assert.Equal(t, "█", bottomRuneLUT[255])
}

func BenchmarkScaledColor(b *testing.B) {
	led := producer.Led{Red: 120, Green: 60, Blue: 30}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = scaledColor(led)
	}
}

func BenchmarkSimulateLedSegment(b *testing.B) {
	p := &TUIPlatform{}
	seg := &segment{
		firstLed: 0,
		lastLed:  99,
		visible:  true,
		leds:     make([]producer.Led, 100),
	}
	for i := range seg.leds {
		seg.leds[i] = producer.Led{Red: float64(i * 2), Green: float64(i), Blue: 50}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = p.simulateLedSegment(seg)
	}
}
