package producer

// The outside interface all concrete producers need to fulfill
type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Start()
	Stop()
	Exit()
	GetIsRunning() bool
}
