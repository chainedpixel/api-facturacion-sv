package checkers

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

const probeEventName = "health.probe"

type domainEventsChecker struct {
	bus    event.Bus
	signal chan string
}

type probeEvent struct {
	id string
	t  time.Time
}

func (e probeEvent) Name() string            { return probeEventName }
func (e probeEvent) OccurredAt() time.Time   { return e.t }
func (e probeEvent) AggregateID() string     { return e.id }
func (e probeEvent) Payload() map[string]any { return map[string]any{"id": e.id} }

type probeHandler struct {
	ch chan string
}

func (h probeHandler) Name() string { return "health_probe" }

func (h probeHandler) Handle(_ context.Context, evt event.Event) error {
	select {
	case h.ch <- evt.AggregateID():
	default:
	}
	return nil
}

func NewDomainEventsChecker(bus event.Bus) health.ComponentChecker {
	ch := make(chan string, 8)
	if bus != nil {
		bus.Subscribe(probeEventName, probeHandler{ch: ch})
	}
	return &domainEventsChecker{bus: bus, signal: ch}
}

func (c *domainEventsChecker) Name() string { return "domain_events" }

func (c *domainEventsChecker) Check() models.Health {
	if c.bus == nil {
		return models.Health{
			Status:  constants.StatusDown,
			Details: utils.TranslateMessage("health.down", "domain_events_not_initialized"),
		}
	}

	id := uuid.NewString()
	c.bus.Publish(context.Background(), probeEvent{id: id, t: time.Now()})

	timeout := time.After(1 * time.Second)
	for {
		select {
		case got := <-c.signal:
			if got == id {
				return models.Health{
					Status:  constants.StatusUp,
					Details: utils.TranslateHealthUp(c.Name()),
				}
			}
		case <-timeout:
			return models.Health{
				Status:  constants.StatusDown,
				Details: utils.TranslateHealthDown(c.Name()),
			}
		}
	}
}
