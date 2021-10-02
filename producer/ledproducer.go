package producer

// The outside interface all concrete producers need to fulfill
type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Fire()
	Exit()
}

// Local Variables:
// compile-command: "cd .. && go build"
// End:
