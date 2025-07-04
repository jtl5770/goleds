package producer

import c "lautenbacher.net/goleds/config"

type Led struct {
	Red   float64
	Green float64
	Blue  float64
}

// True if all components are zero, false otherwise
func (s *Led) IsEmpty() bool {
	return s.Red == 0 && s.Green == 0 && s.Blue == 0
}

// Return a Led with per component the max value of the caller and the
// Led input parameter
func (s *Led) Max(in Led) Led {
	if s.Red > in.Red {
		in.Red = s.Red
	}
	if s.Green > in.Green {
		in.Green = s.Green
	}
	if s.Blue > in.Blue {
		in.Blue = s.Blue
	}
	return in
}

func CombineLeds(allLedRanges map[string][]Led) []Led {
	sumLeds := make([]Led, c.CONFIG.Hardware.Display.LedsTotal)
	for _, currleds := range allLedRanges {
		for j := range currleds {
			sumLeds[j] = currleds[j].Max(sumLeds[j])
		}
	}
	return sumLeds
}
