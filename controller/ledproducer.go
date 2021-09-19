package ledcontroller

type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Fire()
}

type Led struct {
	Red   byte
	Green byte
	Blue  byte
}

func (s Led) IsEmpty() bool {
	return s.Red == 0 && s.Green == 0 && s.Blue == 0
}

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
