package event

import "context"

type Handler interface {
	Name() string
	Handle(ctx context.Context, evt Event) error
}

type Bus interface {
	Publish(ctx context.Context, evt Event)
	Subscribe(eventName string, handler Handler)
}
