package handlers

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/application/handlers/notification"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	test "github.com/chainedpixel/ordo-factus/tests"
)

type fakeMailer struct {
	calls   atomic.Int32
	subject string
	html    string
	plain   string
	err     error
}

func (m *fakeMailer) SendAdminAlert(_ context.Context, subject, html, plain string) error {
	m.calls.Add(1)
	m.subject = subject
	m.html = html
	m.plain = plain
	return m.err
}

type fakeCooldown struct {
	allow bool
	err   error
	calls atomic.Int32
}

func (c *fakeCooldown) ShouldNotify(context.Context, string, string) (bool, error) {
	c.calls.Add(1)
	return c.allow, c.err
}

type fakeRenderer struct {
	subject string
	html    string
	plain   string
	err     error
}

func (r *fakeRenderer) Render(event.Event) (string, string, string, error) {
	return r.subject, r.html, r.plain, r.err
}

func sampleEvent() event.Event {
	return event.ContingencyActivatedEvent{
		BranchID:        1,
		ContingencyType: "1",
		Reason:          "x",
		AffectedDocs:    1,
		OccurredAtTime:  time.Now(),
	}
}

func TestSendsWhenCooldownAllows(t *testing.T) {
	test.TestMain(t)

	mailer := &fakeMailer{}
	cd := &fakeCooldown{allow: true}
	rend := &fakeRenderer{subject: "S", html: "<b>H</b>", plain: "P"}
	h := notification.NewAdminEmailHandler(mailer, cd, rend)

	if err := h.Handle(context.Background(), sampleEvent()); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if mailer.calls.Load() != 1 {
		t.Fatalf("mailer should have been called once, got %d", mailer.calls.Load())
	}
	if mailer.subject != "S" || mailer.html != "<b>H</b>" || mailer.plain != "P" {
		t.Errorf("mailer received unexpected payload: %#v", mailer)
	}
}

func TestSkipsWhenCooldownActive(t *testing.T) {
	test.TestMain(t)

	mailer := &fakeMailer{}
	cd := &fakeCooldown{allow: false}
	rend := &fakeRenderer{subject: "S"}
	h := notification.NewAdminEmailHandler(mailer, cd, rend)

	if err := h.Handle(context.Background(), sampleEvent()); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if mailer.calls.Load() != 0 {
		t.Fatalf("mailer should not have been called, got %d", mailer.calls.Load())
	}
}

func TestProceedsWhenCooldownErrors(t *testing.T) {
	test.TestMain(t)

	mailer := &fakeMailer{}
	cd := &fakeCooldown{allow: false, err: errors.New("redis down")}
	rend := &fakeRenderer{subject: "S"}
	h := notification.NewAdminEmailHandler(mailer, cd, rend)

	if err := h.Handle(context.Background(), sampleEvent()); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if mailer.calls.Load() != 1 {
		t.Fatalf("mailer should have been called when cooldown errors, got %d", mailer.calls.Load())
	}
}

func TestRendererErrorPropagated(t *testing.T) {
	test.TestMain(t)

	mailer := &fakeMailer{}
	cd := &fakeCooldown{allow: true}
	rend := &fakeRenderer{err: errors.New("template broken")}
	h := notification.NewAdminEmailHandler(mailer, cd, rend)

	if err := h.Handle(context.Background(), sampleEvent()); err == nil {
		t.Fatal("expected render error to propagate")
	}
	if mailer.calls.Load() != 0 {
		t.Fatal("mailer should not be called when render fails")
	}
}

func TestNilCooldownAlwaysSends(t *testing.T) {
	test.TestMain(t)

	mailer := &fakeMailer{}
	rend := &fakeRenderer{subject: "S", html: "h", plain: "p"}
	h := notification.NewAdminEmailHandler(mailer, nil, rend)
	for i := 0; i < 3; i++ {
		if err := h.Handle(context.Background(), sampleEvent()); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}
	if mailer.calls.Load() != 3 {
		t.Fatalf("expected 3 sends, got %d", mailer.calls.Load())
	}
}
