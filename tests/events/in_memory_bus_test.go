package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/events"
	test "github.com/chainedpixel/ordo-factus/tests"
)

type fakeEvent struct{ name string }

func (e fakeEvent) Name() string            { return e.name }
func (e fakeEvent) OccurredAt() time.Time   { return time.Unix(0, 0) }
func (e fakeEvent) AggregateID() string     { return "agg" }
func (e fakeEvent) Payload() map[string]any { return map[string]any{"x": 1} }

type persistableEvent struct {
	fakeEvent
	userID, branchID uint
}

func (p persistableEvent) UserID() uint   { return p.userID }
func (p persistableEvent) BranchID() uint { return p.branchID }

type recordingHandler struct {
	name    string
	count   atomic.Int32
	last    chan event.Event
	failNow atomic.Bool
}

func newRecordingHandler(name string) *recordingHandler {
	return &recordingHandler{name: name, last: make(chan event.Event, 4)}
}

func (h *recordingHandler) Name() string { return h.name }

func (h *recordingHandler) Handle(_ context.Context, evt event.Event) error {
	h.count.Add(1)
	h.last <- evt
	if h.failNow.Load() {
		return errors.New("boom")
	}
	return nil
}

type panickingHandler struct{ name string }

func (h panickingHandler) Name() string                              { return h.name }
func (h panickingHandler) Handle(context.Context, event.Event) error { panic("kaboom") }

type recordingRepo struct {
	mu      sync.Mutex
	saved   int
	lastEvt string
}

func (r *recordingRepo) Save(_ context.Context, evtType string, _, _ uint, _ string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved++
	r.lastEvt = evtType
	return nil
}

func TestPublishDispatchesAllSubscribers(t *testing.T) {
	test.TestMain(t)

	bus := events.NewInMemoryBus(nil)
	h1 := newRecordingHandler("h1")
	h2 := newRecordingHandler("h2")
	bus.Subscribe("e", h1)
	bus.Subscribe("e", h2)

	bus.Publish(context.Background(), fakeEvent{name: "e"})

	for _, h := range []*recordingHandler{h1, h2} {
		select {
		case <-h.last:
		case <-time.After(time.Second):
			t.Fatalf("handler %s not invoked", h.name)
		}
	}
}

func TestPublishOnlyToMatchingName(t *testing.T) {
	test.TestMain(t)

	bus := events.NewInMemoryBus(nil)
	other := newRecordingHandler("other")
	target := newRecordingHandler("target")
	bus.Subscribe("a", other)
	bus.Subscribe("b", target)

	bus.Publish(context.Background(), fakeEvent{name: "b"})

	select {
	case <-target.last:
	case <-time.After(time.Second):
		t.Fatal("target handler not invoked")
	}
	if other.count.Load() != 0 {
		t.Fatalf("other handler should not have fired, got %d", other.count.Load())
	}
}

func TestPanicInOneHandlerDoesNotBlockOthers(t *testing.T) {
	test.TestMain(t)

	bus := events.NewInMemoryBus(nil)
	bus.Subscribe("e", panickingHandler{name: "boom"})
	good := newRecordingHandler("good")
	bus.Subscribe("e", good)

	bus.Publish(context.Background(), fakeEvent{name: "e"})

	select {
	case <-good.last:
	case <-time.After(time.Second):
		t.Fatal("good handler did not run after sibling panic")
	}
}

func TestPersistsOnlyPersistableEvents(t *testing.T) {
	test.TestMain(t)

	repo := &recordingRepo{}
	bus := events.NewInMemoryBus(repo)
	h := newRecordingHandler("h")
	bus.Subscribe("p", h)
	bus.Subscribe("np", h)

	bus.Publish(context.Background(), persistableEvent{
		fakeEvent: fakeEvent{name: "p"},
		userID:    1,
		branchID:  2,
	})
	bus.Publish(context.Background(), fakeEvent{name: "np"})

	deadline := time.After(time.Second)
	for h.count.Load() < 2 {
		select {
		case <-h.last:
		case <-deadline:
			t.Fatalf("handlers did not finish: %d", h.count.Load())
		}
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.saved != 1 {
		t.Fatalf("expected 1 persisted event, got %d", repo.saved)
	}
	if repo.lastEvt != "p" {
		t.Fatalf("expected last persisted event 'p', got %q", repo.lastEvt)
	}
}
