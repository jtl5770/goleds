package producer

type Led struct {
	Red   float64
	Green float64
	Blue  float64
}

// True if all components are zero, false otherwise
func (s *Led) IsEmpty() bool {
	return s.Red == 0 && s.Green == 0 && s.Blue == 0
}

func CombineLeds(allLedRanges map[string][]Led, target []Led) {
	clear(target)

	for _, currleds := range allLedRanges {
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
