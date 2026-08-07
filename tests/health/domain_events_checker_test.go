package health

import (
	"strings"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/events"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/health/checkers"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func TestDomainEventsChecker_UpWhenBusDelivers(t *testing.T) {
	test.TestMain(t)

	bus := events.NewInMemoryBus(nil)
	c := checkers.NewDomainEventsChecker(bus)

	got := c.Check()
	if got.Status != constants.StatusUp {
		t.Fatalf("expected UP, got %q (details=%q)", got.Status, got.Details)
	}
	if !strings.Contains(strings.ToLower(got.Details), "eventos") &&
		!strings.Contains(strings.ToLower(got.Details), "events") {
		t.Errorf("details should reference events, got %q", got.Details)
	}
}

func TestDomainEventsChecker_DownWhenBusNil(t *testing.T) {
	test.TestMain(t)

	c := checkers.NewDomainEventsChecker(nil)
	got := c.Check()
	if got.Status != constants.StatusDown {
		t.Fatalf("expected DOWN, got %q", got.Status)
	}
	if !strings.Contains(strings.ToLower(got.Details), "no inicializado") &&
		!strings.Contains(strings.ToLower(got.Details), "not initialized") {
		t.Errorf("details should mention bus not initialized, got %q", got.Details)
	}
}

func TestDomainEventsChecker_Name(t *testing.T) {
	c := checkers.NewDomainEventsChecker(nil)
	if c.Name() != "domain_events" {
		t.Fatalf("name = %q", c.Name())
	}
}
