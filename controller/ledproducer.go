package ledcontroller

type LedProducer interface {
	GetLeds() []Led
	GetUID() int
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
