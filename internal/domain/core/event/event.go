package event

import "time"

type Event interface {
	Name() string
	OccurredAt() time.Time
	AggregateID() string
	Payload() map[string]any
}
