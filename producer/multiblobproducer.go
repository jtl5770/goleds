package producer

import (
	"fmt"
	"log"
	"math"
	"time"
	t "time"

	c "lautenbacher.net/goleds/config"
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

func NewBlob(uid string) *Blob {
	inst := Blob{
		uid: uid,
		led: Led{
			Red:   c.CONFIG.MultiBlobLED.BlobCfg[uid].LedRGB[0],
			Green: c.CONFIG.MultiBlobLED.BlobCfg[uid].LedRGB[1],
			Blue:  c.CONFIG.MultiBlobLED.BlobCfg[uid].LedRGB[2],
		},
		last_x: c.CONFIG.MultiBlobLED.BlobCfg[uid].X,
		x:      c.CONFIG.MultiBlobLED.BlobCfg[uid].X,
		width:  c.CONFIG.MultiBlobLED.BlobCfg[uid].Width,
		delta:  c.CONFIG.MultiBlobLED.BlobCfg[uid].DeltaX,
	}
	if inst.delta < 0 {
		inst.dir = -1
	} else {
		inst.dir = 1
	}
	inst.delta = math.Abs(inst.delta)
	return &inst
}

func (s *Blob) getBlobLeds() []Led {
	leds := make([]Led, c.CONFIG.Hardware.Display.LedsTotal)

	for i := 0; i < c.CONFIG.Hardware.Display.LedsTotal; i++ {
		y := math.Exp(-1 * (math.Pow(float64(i)-s.x, 2) / s.width))
		leds[i] = Led{
			byte(math.Round(float64(s.led.Red) * y)),
			byte(math.Round(float64(s.led.Green) * y)),
			byte(math.Round(float64(s.led.Blue) * y)),
		}
	}
	return leds
}

func (s *Blob) switchDirection() {
	s.dir = s.dir * -1
}

type MultiBlobProducer struct {
	*AbstractProducer
	allblobs map[string]*Blob
}

func NewMultiBlobProducer(uid string, ledsChanged chan LedProducer) *MultiBlobProducer {
	inst := MultiBlobProducer{
		AbstractProducer: NewAbstractProducer(uid, ledsChanged),
	}
	inst.runfunc = inst.runner

	inst.allblobs = make(map[string]*Blob)
	for uid := range c.CONFIG.MultiBlobLED.BlobCfg {
		blob := NewBlob(uid)
		inst.allblobs[uid] = blob
	}
	return &inst
}

func (s *MultiBlobProducer) runner(startTime t.Time) {
	triggerduration := time.NewTicker(c.CONFIG.MultiBlobLED.Duration)
	tick := time.NewTicker(c.CONFIG.MultiBlobLED.Delay)
	defer func() {
		s.updateMutex.Lock()
		s.isRunning = false
		s.updateMutex.Unlock()
		tick.Stop()
		triggerduration.Stop()
	}()

	// directly stop the Trigger to control the duration of the effect
	// whenever it is configured to run all the time (and not only
	// MultiBlobLED.Duration after a trigger)
	if !c.CONFIG.MultiBlobLED.Trigger {
		triggerduration.Stop()
	}

	for {
		select {
		case <-triggerduration.C:
			for i := 0; i < c.CONFIG.Hardware.Display.LedsTotal; i++ {
				s.setLed(i, Led{})
			}
			s.ledsChanged <- s
			return
		case <-s.stop:
			log.Println("Stopped MultiBlobProducer...")
			return
		case <-tick.C:
			// compute new x value
			for _, blob := range s.allblobs {
				blob.x = blob.x + (blob.delta * blob.dir)
			}

			// detect & handle collision
			detectAndHandleCollisions(s.allblobs)

			// push update event for Leds
			tmp := make(map[string][]Led)
			for _, blob := range s.allblobs {
				tmp[blob.uid] = blob.getBlobLeds()
			}
			combined := CombineLeds(tmp)
			for i := 0; i < c.CONFIG.Hardware.Display.LedsTotal; i++ {
				s.setLed(i, combined[i])
			}
			s.ledsChanged <- s

			// update last_x value to current x
			for _, blob := range s.allblobs {
				blob.last_x = blob.x
			}
		}
	}
}

func detectAndHandleCollisions(blobs map[string]*Blob) {
	max := float64(c.CONFIG.Hardware.Display.LedsTotal)
	var checkinter []*Blob
	collblobs := make(map[string]*Blob)

	for _, blob := range blobs {
		if ((blob.x > max) && (blob.dir > 0)) ||
			((blob.x < 0) && (blob.dir < 0)) {
			// log.Println(fmt.Sprintf("%s hit boundary. x=%f ", blob.uid, blob.x))
			blob.switchDirection()
			collblobs[blob.uid] = blob
		} else {
			// we will look only for collisions between blobs which are not right now also hitting the stripe boundaries
			// to make sure that a blob colliding with the boundary always changes direction away from the boundary
			checkinter = append(checkinter, blob)
		}
	}

	size := len(checkinter)
	if size >= 2 {
		for i := 0; i < size; i++ {
			blob_a := checkinter[i]
			for j := i + 1; j < size; j++ {
				blob_b := checkinter[j]
				if detectBlobColl(blob_a, blob_b) {
					collblobs[blob_a.uid] = blob_a
					collblobs[blob_b.uid] = blob_b
				}
			}
		}
		// for all blobs that take part in a collision we set their x value back to the last know value
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
		// log.Println("Collision detected between " + blob_a.uid + " and " + blob_b.uid)
		var left *Blob
		var right *Blob

		// find out which one is the "left one" and which is the "right one", to simplify handling
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
			// chsing from right to left - only right changes direction
			// log.Println(fmt.Sprintf("Chasing R2L: %s - Direction %f  |  %s - Direction %f", left.uid, left.dir, right.uid, right.dir))
			right.switchDirection()
		} else if left.dir < 0 && right.dir > 0 {
			log.Println(fmt.Sprintf("%s - Direction %f  | %s - Direction %f", left.uid, left.dir, right.uid, right.dir))
			log.Println("Caution: colliding blobs " + left.uid + " and " + right.uid +
				" are already heading in opposite directions")
		}
	}
	return collide
}
