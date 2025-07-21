package util

import "time"

// Trigger represents a sensor event.
type Trigger struct {
	ID        string
	Value     int
	Timestamp time.Time
}

// NewTrigger creates a new Trigger instance.
func NewTrigger(id string, value int, time time.Time) *Trigger {
	inst := Trigger{
		ID:        id,
		Value:     value,
		Timestamp: time,
	}
	return &inst
}
