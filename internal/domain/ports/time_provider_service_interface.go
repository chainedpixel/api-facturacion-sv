package ports

import "time"

// TimeProvider is an interface that defines the methods that a time provider must implement
type TimeProvider interface {
	Now() time.Time
	Sleep(d time.Duration)
}
