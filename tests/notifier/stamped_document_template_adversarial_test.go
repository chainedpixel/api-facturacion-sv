package notifier

import (
	"strings"
	"testing"
	"time"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/event"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/notifier/email"
)

// TestAdversarialStampedDocumentTemplate attacks the stamped-not-persisted email with empty fields and hostile content.
func TestAdversarialStampedDocumentTemplate(t *testing.T) {
	setupRendererConfig(t)
	renderer, err := email.NewTemplateRenderer()
	if err != nil {
		t.Fatalf("renderer: %v", err)
	}

	t.Run("subject falls back to the generation code without control number", func(t *testing.T) {
		subject, _, _, err := renderer.Render(event.StampedDocumentNotPersistedEvent{GenerationCode: "GEN-ONLY", OccurredAtTime: time.Now()})
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if !strings.Contains(subject, "GEN-ONLY") || strings.Contains(subject, "%!") || strings.Contains(subject, "%s") {
			t.Errorf("unexpected subject %q", subject)
		}
	})

	t.Run("all fields empty still renders", func(t *testing.T) {
		subject, html, plain, err := renderer.Render(event.StampedDocumentNotPersistedEvent{})
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if strings.Contains(subject, "%!") || html == "" || plain == "" {
			t.Errorf("empty event must render, subject=%q", subject)
		}
	})

	t.Run("html content is escaped", func(t *testing.T) {
		hostile := `<script>alert("x")</script>`
		_, html, plain, err := renderer.Render(event.StampedDocumentNotPersistedEvent{
			ControlNumber: "DTE-01-X", GenerationCode: "G", LastError: hostile, ReceptionStamp: hostile, OccurredAtTime: time.Now(),
		})
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if strings.Contains(html, "<script>alert") {
			t.Error("html body must escape the error text")
		}
		if !strings.Contains(plain, hostile) {
			t.Error("plain text keeps the raw text")
		}
	})

	t.Run("unknown dte type renders raw code", func(t *testing.T) {
		_, html, _, err := renderer.Render(event.StampedDocumentNotPersistedEvent{DTEType: "99", GenerationCode: "G", OccurredAtTime: time.Now()})
		if err != nil || !strings.Contains(html, "99") {
			t.Errorf("unknown type must render, err=%v", err)
		}
	})
}
