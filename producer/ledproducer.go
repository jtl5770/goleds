package producer

import u "lautenbacher.net/goleds/util"

// The outside interface all concrete producers need to fulfill
type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Start(trigger *u.Trigger)
	Stop()
	Exit()
	GetIsRunning() bool
}
