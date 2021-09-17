package ledcontroller

type LedProducer interface {
	GetLeds() []Led
	GetUID() string
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
