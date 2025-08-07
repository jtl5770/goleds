package producer

import (
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	c "lautenbacher.net/goleds/config"
	u "lautenbacher.net/goleds/util"
)

type Blob struct {
	uid    string
	led    Led
	last_x float64
	x      float64
	width  float64
	delta  float64
	dir    float64
}

func NewBlob(uid string, ledRGB []float64, x, width, deltaX float64) *Blob {
	inst := Blob{
		uid: uid,
		led: Led{
			Red:   ledRGB[0],
			Green: ledRGB[1],
			Blue:  ledRGB[2],
		},
		last_x: x,
		x:      x,
		width:  width,
		delta:  deltaX,
	}
	if inst.delta < 0 {
		inst.dir = -1
	} else {
		inst.dir = 1
	}
	inst.delta = math.Abs(inst.delta)
	return &inst
}

// applyTo calculates the blob's contribution and adds it to an existing LED slice
// using a Max function to blend.
func (s *Blob) applyTo(leds []Led) {
	ledsTotal := len(leds)
	// Optimization: only calculate for LEDs that will be visibly affected.
	// The intensity of the blob is based on a Gaussian function. We can determine
	// a bounding box outside of which the light intensity is negligible.
	// `y = exp(-distance_squared / s.width)`. If we say `y < 0.01` is negligible,
	// this corresponds to `distance_squared / s.width > 4.6`.
	// We use `5.0` for a rounder number and to be safe.
	bound := int(math.Ceil(math.Sqrt(5.0 * s.width)))
	start := int(math.Floor(s.x)) - bound
	if start < 0 {
		start = 0
	}
	end := int(math.Ceil(s.x)) + bound
	if end >= ledsTotal {
		end = ledsTotal
	}

	for i := start; i < end; i++ {
		y := math.Exp(-1 * (math.Pow(float64(i)-s.x, 2) / s.width))
		// No need to check for small y here, the loop bounds already handle it.
		blobLed := Led{s.led.Red * y, s.led.Green * y, s.led.Blue * y}
		leds[i] = leds[i].Max(blobLed)
	}
}

func (s *Blob) switchDirection() {
	s.dir = s.dir * -1
}

type MultiBlobProducer struct {
	*AbstractProducer
	allblobs map[string]*Blob
	duration time.Duration
	delay    time.Duration
}

func NewMultiBlobProducer(uid string, ledsChanged *u.AtomicMapEvent[LedProducer], ledsTotal int, duration, delay time.Duration, blobCfg map[string]c.BlobCfg, endwg *sync.WaitGroup) *MultiBlobProducer {
	inst := &MultiBlobProducer{
		duration: duration,
		delay:    delay,
	}
	inst.AbstractProducer = NewAbstractProducer(uid, ledsChanged, inst.runner, ledsTotal)
	if endwg != nil {
		inst.AbstractProducer.endWg = endwg
	}

	inst.allblobs = make(map[string]*Blob)
	for uid, cfg := range blobCfg {
		blob := NewBlob(uid, cfg.LedRGB, cfg.X, cfg.Width, cfg.DeltaX)
		inst.allblobs[uid] = blob
	}
	return inst
}

func (s *MultiBlobProducer) fade_in_or_out(fadein bool) {
	intervals := 20
	delay := 20 * time.Millisecond
	// The pattern to be faded is in s.leds, but we want a stable base
	// status at the beginning of the animation
	s.ledsMutex.RLock()
	baseLeds := make([]Led, len(s.leds))
	copy(baseLeds, s.leds)
	s.ledsMutex.RUnlock()

	for counter := 0; counter <= intervals; counter++ {
		var step int
		if fadein {
			step = counter
		} else {
			step = intervals - counter
		}

		factor := float64(step) / float64(intervals)

		s.ledsMutex.Lock()
		for i, led := range baseLeds {
			// Directly manipulate s.leds to avoid overhead of setLed
			if i < len(s.leds) {
				s.leds[i] = Led{led.Red * factor, led.Green * factor, led.Blue * factor}
			}
		}
		s.ledsMutex.Unlock()
		s.ledsChanged.Send(s.GetUID(), s) // Send one notification per fade step
		time.Sleep(delay)
	}
}

func (s *MultiBlobProducer) runner() {
	triggerduration := time.NewTicker(s.duration)
	tick := time.NewTicker(s.delay)
	countup_run := false
	defer func() {
		tick.Stop()
		triggerduration.Stop()
	}()

	for {
		select {
		case <-triggerduration.C:
			// Doing the fadeout after the time is up
			s.fade_in_or_out(false)
			return
		case <-s.stopchan:
			// Doing the fadeout when Stop() is triggered
			s.fade_in_or_out(false)
			return
		case <-tick.C:
			// compute new x value
			for _, blob := range s.allblobs {
				blob.x = blob.x + (blob.delta * blob.dir)
			}

			// detect & handle collision
			detectAndHandleCollisions(s.allblobs, len(s.leds))

			// push update event for Leds
			s.ledsMutex.Lock()
			// clear slice
			for i := range s.leds {
				s.leds[i] = Led{}
			}
			// combine blobs by applying each one to the producer's led slice
			for _, blob := range s.allblobs {
				blob.applyTo(s.leds)
			}
			s.ledsMutex.Unlock()

			if countup_run {
				s.ledsChanged.Send(s.GetUID(), s)
			} else {
				// The "countup" similar to the "countdown" fade out but fade in
				// at the start of the blob period
				s.fade_in_or_out(true)
				countup_run = true
			}
			// update last_x value to current x
			for _, blob := range s.allblobs {
				blob.last_x = blob.x
			}
		}
	}
}

func detectAndHandleCollisions(blobs map[string]*Blob, ledsTotal int) {
	max := float64(ledsTotal)
	var checkinter []*Blob
	collblobs := make(map[string]*Blob)

	for _, blob := range blobs {
		if ((blob.x > max) && (blob.dir > 0)) ||
			((blob.x < 0) && (blob.dir < 0)) {
			// log.Println(fmt.Sprintf("%s hit boundary. x=%f ", blob.uid, blob.x))
			blob.switchDirection()
			collblobs[blob.uid] = blob
		} else {
			// we will look only for collisions between blobs which
			// are not right now also hitting the stripe boundaries to
			// make sure that a blob colliding with the boundary
			// always changes direction away from the boundary
			checkinter = append(checkinter, blob)
		}
	}

	size := len(checkinter)
	if size >= 2 {
		for i := range size {
			blob_a := checkinter[i]
			for j := i + 1; j < size; j++ {
				blob_b := checkinter[j]
				if detectBlobColl(blob_a, blob_b) {
					collblobs[blob_a.uid] = blob_a
					collblobs[blob_b.uid] = blob_b
				}
			}
		}
		// for all blobs that take part in a collision we set their x
		// value back to the last know value
		for _, blob := range collblobs {
			blob.x = blob.last_x
		}
	}
}

func detectBlobColl(blob_a *Blob, blob_b *Blob) bool {
	a1, a2 := blob_a.x, blob_a.last_x
	a_start := math.Min(a1, a2)
	a_end := math.Max(a1, a2)
	b1, b2 := blob_b.x, blob_b.last_x
	b_start := math.Min(b1, b2)
	b_end := math.Max(b1, b2)

	collide := (a_start <= b_end) && (b_start <= a_end)
	if collide {
		// Generate a random value between 0 and 1 and if smaller
		// than 0.2, just return as if no collision is detected.
		if rand.Float64() < 0.33 {
			return false
		}
		// log.Println("Collision detected between " + blob_a.uid + " and " + blob_b.uid)
		var left *Blob
		var right *Blob

		// find out which one is the "left one" and which is the
		// "right one", to simplify handling
		if blob_a.last_x < blob_b.last_x {
			left = blob_a
			right = blob_b
		} else {
			left = blob_b
			right = blob_a
		}

		if left.dir > 0 && right.dir < 0 {
			// heading straight at each other
			// log.Println(fmt.Sprintf("Head2Head: %s - Direction %f  |  %s - Direction %f", left.uid, left.dir, right.uid, right.dir))
			left.switchDirection()
			right.switchDirection()
		} else if left.dir > 0 && right.dir > 0 {
			// chasing from left to right - only left changes direction
			// log.Println(fmt.Sprintf("Chasing L2R: %s - Direction %f  |  %s - Direction %f", left.uid, left.dir, right.uid, right.dir))
			left.switchDirection()
		} else if left.dir < 0 && right.dir < 0 {
			// chasing from right to left - only right changes direction
			// log.Println(fmt.Sprintf("Chasing R2L: %s - Direction %f  |  %s - Direction %f", left.uid, left.dir, right.uid, right.dir))
			right.switchDirection()
		} else if left.dir < 0 && right.dir > 0 {
			slog.Warn("Colliding blobs are already heading in opposite directions", "left_uid", left.uid, "left_dir", left.dir, "right_uid", right.uid, "right_dir", right.dir)
		}
	}
	return collide
}
