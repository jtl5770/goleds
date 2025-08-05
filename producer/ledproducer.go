package producer

import u "lautenbacher.net/goleds/util"

// The outside interface all concrete producers need to fulfill
type LedProducer interface {
	GetLeds() []Led
	GetUID() string
	Start()
	SendTrigger(trigger *u.Trigger)
	TryStop() (bool, error)
	Exit()
}
