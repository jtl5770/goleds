package ledcontroller

type LedProducer interface {
	GetLeds() []Led
	GetUID() int
}
