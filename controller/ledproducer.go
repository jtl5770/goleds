package ledcontroller

type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Fire()
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
