package producer

type Led struct {
	Red   byte
	Green byte
	Blue  byte
}

// True if all components are zero, false otherwise
func (s Led) IsEmpty() bool {
	return s.Red == 0 && s.Green == 0 && s.Blue == 0
}

// Return a Led with per component the max value of the caller and the
// in parameter
func (s Led) Max(in Led) Led {
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

// Local Variables:
// compile-command: "cd .. && go build"
// End:
