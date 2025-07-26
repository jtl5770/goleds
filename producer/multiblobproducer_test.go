package producer

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

func TestNewMultiBlobProducer(t *testing.T) {
	ledsChanged := u.NewAtomicEvent[LedProducer]()
	ledsTotal := 10
	duration := 5 * time.Second
	delay := 50 * time.Millisecond
	blobCfg := map[string]c.BlobCfg{
		"blob1": {DeltaX: 0.1, X: 2.0, Width: 1.0, LedRGB: []float64{255, 0, 0}},
		"blob2": {DeltaX: -0.2, X: 8.0, Width: 1.5, LedRGB: []float64{0, 255, 0}},
	}

	p := NewMultiBlobProducer("test_multiblob", ledsChanged, ledsTotal, duration, delay, blobCfg, nil)

	assert.Equal(t, "test_multiblob", p.GetUID())
	assert.Len(t, p.leds, ledsTotal)
	assert.Equal(t, duration, p.duration)
	assert.Equal(t, delay, p.delay)
	assert.Len(t, p.allblobs, 2)

	// Verify individual blobs
	blob1 := p.allblobs["blob1"]
	assert.NotNil(t, blob1)
	assert.Equal(t, "blob1", blob1.uid)
	assert.Equal(t, Led{Red: 255, Green: 0, Blue: 0}, blob1.led)
	assert.Equal(t, 2.0, blob1.x)
	assert.Equal(t, 1.0, blob1.width)
	assert.Equal(t, 0.1, blob1.delta)
	assert.Equal(t, float64(1), blob1.dir)

	blob2 := p.allblobs["blob2"]
	assert.NotNil(t, blob2)
	assert.Equal(t, "blob2", blob2.uid)
	assert.Equal(t, Led{Red: 0, Green: 255, Blue: 0}, blob2.led)
	assert.Equal(t, 8.0, blob2.x)
	assert.Equal(t, 1.5, blob2.width)
	assert.Equal(t, 0.2, blob2.delta)
	assert.Equal(t, float64(-1), blob2.dir)

	assert.False(t, p.GetIsRunning())
}

func TestBlob_getBlobLeds(t *testing.T) {
	blob := NewBlob("test_blob", []float64{255, 0, 0}, 5.0, 1.0, 0.0)
	ledsTotal := 10

	leds := blob.getBlobLeds(ledsTotal)

	assert.Len(t, leds, ledsTotal)

	// Check the peak at x=5.0
	assert.InDelta(t, 255.0, leds[5].Red, 0.001)
	assert.InDelta(t, 0.0, leds[5].Green, 0.001)
	assert.InDelta(t, 0.0, leds[5].Blue, 0.001)

	// Check a point further away (e.g., x=4.0 or x=6.0, which should be equal)
	assert.InDelta(t, 255.0*math.Exp(-1.0), leds[4].Red, 0.001) // exp(-1*(4-5)^2/1) = exp(-1)
	assert.InDelta(t, 255.0*math.Exp(-1.0), leds[6].Red, 0.001)

	// Check a point far away (e.g., x=0.0)
	assert.InDelta(t, 255.0*math.Exp(-25.0), leds[0].Red, 0.001)
}

func TestBlob_switchDirection(t *testing.T) {
	blob := NewBlob("test_blob", []float64{255, 0, 0}, 5.0, 1.0, 0.1)
	initialDir := blob.dir

	blob.switchDirection()
	assert.Equal(t, -initialDir, blob.dir)

	blob.switchDirection()
	assert.Equal(t, initialDir, blob.dir)
}

func TestDetectAndHandleCollisions_Boundary(t *testing.T) {
	ledsTotal := 10

	// Test blob hitting right boundary
	blobRight := NewBlob("blobRight", []float64{255, 0, 0}, float64(ledsTotal)+1, 1.0, 0.1)
	blobRight.dir = 1 // Moving right
	blobs := map[string]*Blob{"blobRight": blobRight}
	detectAndHandleCollisions(blobs, ledsTotal)
	assert.Equal(t, float64(-1), blobRight.dir, "Blob hitting right boundary should reverse direction")
	assert.Equal(t, float64(ledsTotal)+1, blobRight.x, "Blob x should be reverted to last_x")

	// Test blob hitting left boundary
	blobLeft := NewBlob("blobLeft", []float64{0, 255, 0}, -1.0, 1.0, -0.1)
	blobLeft.dir = -1 // Moving left
	blobs = map[string]*Blob{"blobLeft": blobLeft}
	detectAndHandleCollisions(blobs, ledsTotal)
	assert.Equal(t, float64(1), blobLeft.dir, "Blob hitting left boundary should reverse direction")
	assert.Equal(t, -1.0, blobLeft.x, "Blob x should be reverted to last_x")

	// Test blob not hitting boundary
	blobNoHit := NewBlob("blobNoHit", []float64{0, 0, 255}, 5.0, 1.0, 0.1)
	blobNoHit.dir = 1
	blobs = map[string]*Blob{"blobNoHit": blobNoHit}
	detectAndHandleCollisions(blobs, ledsTotal)
	assert.Equal(t, float64(1), blobNoHit.dir, "Blob not hitting boundary should not change direction")
	assert.Equal(t, 5.0, blobNoHit.x, "Blob x should not be reverted")
}
