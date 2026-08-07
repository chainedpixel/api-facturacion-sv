package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/notifier/email"
	apihandlers "github.com/chainedpixel/ordo-factus/internal/infrastructure/api/handlers"
	test "github.com/chainedpixel/ordo-factus/tests"
)

type captureMailer struct {
	calls   atomic.Int32
	subject string
	html    string
	plain   string
	err     error
}

func (m *captureMailer) SendAdminAlert(_ context.Context, subject, html, plain string) error {
	m.calls.Add(1)
	m.subject = subject
	m.html = html
	m.plain = plain
	return m.err
}

func newDebugHandler(t *testing.T, mailer *captureMailer) *apihandlers.DebugNotifyHandler {
	t.Helper()
	test.TestMain(t)
	config.Server.APIVersion = "3.0.0"
	r, err := email.NewTemplateRenderer()
	if err != nil {
		t.Fatalf("renderer: %v", err)
	}
	return apihandlers.NewDebugNotifyHandler(mailer, r)
}

func TestDebugNotify_DefaultEventIsContingency(t *testing.T) {
	mailer := &captureMailer{}
	h := newDebugHandler(t, mailer)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/debug/notify-test", nil)
	h.SendTestAlert(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if mailer.calls.Load() != 1 {
		t.Fatalf("mailer should be called once, got %d", mailer.calls.Load())
	}
	if !strings.Contains(mailer.subject, "Contingencia") {
		t.Errorf("subject should be contingency, got %q", mailer.subject)
	}
	if !strings.Contains(mailer.html, "API v3.0.0") {
		t.Error("html should include API v3.0.0")
	}
}

func TestDebugNotify_EmissionVariant(t *testing.T) {
	mailer := &captureMailer{}
	h := newDebugHandler(t, mailer)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/debug/notify-test?event=emission", nil)
	h.SendTestAlert(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(mailer.subject, "DTE-01-0001-000000099") {
		t.Errorf("subject should reference sample ControlNumber, got %q", mailer.subject)
	}
}

func TestDebugNotify_JobVariant(t *testing.T) {
	mailer := &captureMailer{}
	h := newDebugHandler(t, mailer)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/debug/notify-test?event=job", nil)
	h.SendTestAlert(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(mailer.subject, "Retransmisión") {
		t.Errorf("subject should reference job retransmission, got %q", mailer.subject)
	}
}

func TestDebugNotify_RespondsServiceUnavailableWhenStackMissing(t *testing.T) {
	test.TestMain(t)
	h := apihandlers.NewDebugNotifyHandler(nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/debug/notify-test", nil)
	h.SendTestAlert(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
