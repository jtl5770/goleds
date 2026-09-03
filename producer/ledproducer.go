package producer

import "lautenbacher.net/goleds/util"

// The outside interface all concrete producers need to fulfill
type LedProducer interface {
	GetLeds(buffer []Led)
	GetUID() string
	GetPriority() int
	IsActive() bool
	Start()
	SendTrigger(trigger *util.Trigger)
	TryStop() (bool, error)
	Exit()
}
