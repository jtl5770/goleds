package producer

import "math"

type Led struct {
	Red   float64
	Green float64
	Blue  float64
}

// True if all components are zero, false otherwise
func (s *Led) IsEmpty() bool {
	return s.Red == 0 && s.Green == 0 && s.Blue == 0
}

func CombineLeds(allLedRanges map[string][]Led, producers map[string]LedProducer, target []Led) {
	clear(target)

	maxPrio := math.MinInt32
	hasActive := false
	for _, prod := range producers {
		if prod.IsActive() {
			hasActive = true
			if prio := prod.GetPriority(); prio > maxPrio {
				maxPrio = prio
			}
		}
	}

	if !hasActive {
		return
	}

	for key, currleds := range allLedRanges {
		if prod, ok := producers[key]; ok {
			if !prod.IsActive() || prod.GetPriority() < maxPrio {
				continue
			}
		}
		n := min(len(currleds), len(target))
		for j := 0; j < n; j++ {
			led := currleds[j]
			t := &target[j]
			if led.Red > t.Red {
				t.Red = led.Red
			}
			if led.Green > t.Green {
				t.Green = led.Green
			}
			if led.Blue > t.Blue {
				t.Blue = led.Blue
			}
		}
	}
}
