package events

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type InMemoryBus struct {
	mu       sync.RWMutex
	handlers map[string][]event.Handler
	repo     event.Repository
}

func NewInMemoryBus(repo event.Repository) *InMemoryBus {
	return &InMemoryBus{
		handlers: make(map[string][]event.Handler),
		repo:     repo,
	}
}

func (b *InMemoryBus) Subscribe(eventName string, handler event.Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

func (b *InMemoryBus) Publish(ctx context.Context, evt event.Event) {
	b.persist(ctx, evt)

	b.mu.RLock()
	hs := append([]event.Handler(nil), b.handlers[evt.Name()]...)
	b.mu.RUnlock()

	for _, h := range hs {
		go b.dispatch(context.WithoutCancel(ctx), h, evt)
	}
}

func (b *InMemoryBus) dispatch(ctx context.Context, h event.Handler, evt event.Event) {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("event handler panic recovered", map[string]interface{}{
				"event":   evt.Name(),
				"handler": h.Name(),
				"panic":   r,
			})
		}
	}()
	if err := h.Handle(ctx, evt); err != nil {
		logs.Error("event handler returned error", map[string]interface{}{
			"event":   evt.Name(),
			"handler": h.Name(),
			"error":   err.Error(),
		})
	}
}

func (b *InMemoryBus) persist(ctx context.Context, evt event.Event) {
	if b.repo == nil {
		return
	}
	pe, ok := evt.(event.PersistableEvent)
	if !ok {
		return
	}
	payload, err := json.Marshal(evt.Payload())
	if err != nil {
		logs.Warn("failed to marshal event payload", map[string]interface{}{
			"event": evt.Name(),
			"error": err.Error(),
		})
		return
	}
	if err := b.repo.Save(ctx, evt.Name(), pe.UserID(), pe.BranchID(), string(payload), evt.OccurredAt()); err != nil {
		logs.Warn("failed to persist domain event", map[string]interface{}{
			"event": evt.Name(),
			"error": err.Error(),
		})
	}
}
