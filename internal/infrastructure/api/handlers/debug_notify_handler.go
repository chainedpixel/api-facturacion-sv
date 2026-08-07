package handlers

import (
	"net/http"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/notification"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/notifier/email"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type DebugNotifyHandler struct {
	mailer         notification.Mailer
	renderer       *email.TemplateRenderer
	responseWriter *response.ResponseWriter
}

func NewDebugNotifyHandler(mailer notification.Mailer, renderer *email.TemplateRenderer) *DebugNotifyHandler {
	return &DebugNotifyHandler{
		mailer:         mailer,
		renderer:       renderer,
		responseWriter: response.NewResponseWriter(),
	}
}

// SendTestAlert renders and sends a sample admin email, bypassing the cooldown.
// Query param: ?event=contingency|emission|job (default contingency).
func (h *DebugNotifyHandler) SendTestAlert(w http.ResponseWriter, r *http.Request) {
	if h.mailer == nil || h.renderer == nil {
		h.responseWriter.Error(w, http.StatusServiceUnavailable, "notification stack not initialized", nil)
		return
	}

	evt := buildSampleEvent(r.URL.Query().Get("event"))
	subject, html, plain, err := h.renderer.Render(evt)
	if err != nil {
		logs.Error("debug notify render failed", map[string]interface{}{"error": err.Error()})
		h.responseWriter.Error(w, http.StatusInternalServerError, "render failed", []string{err.Error()})
		return
	}

	if err := h.mailer.SendAdminAlert(r.Context(), subject, html, plain); err != nil {
		logs.Error("debug notify send failed", map[string]interface{}{"error": err.Error()})
		h.responseWriter.Error(w, http.StatusInternalServerError, "send failed", []string{err.Error()})
		return
	}

	h.responseWriter.Success(w, http.StatusOK, map[string]any{
		"event":   evt.Name(),
		"subject": subject,
		"sent":    true,
	}, nil)
}

func buildSampleEvent(kind string) event.Event {
	now := time.Now()
	switch kind {
	case "emission":
		return event.EmissionFailureEvent{
			BranchID:             1,
			BranchAddress:        "San Salvador, San Salvador – Calle Principal #123",
			NIT:                  "00000000000000",
			ClientName:           "Empresa de Prueba S.A. de C.V.",
			ControlNumber:        "DTE-01-0001-000000099",
			GenerationCode:       "DEBUG-SAMPLE",
			DTEType:              "01",
			ErrorCode:            "MH-DEBUG",
			LastError:            "Documento no válido según esquema JSON de Hacienda",
			HaciendaObservations: []string{"El campo emisor.nit no cumple el formato requerido", "El campo receptor.nombre excede la longitud máxima permitida"},
			Attempts:             3,
			OccurredAtTime:       now,
		}
	case "job":
		return event.RetransmissionJobFailedEvent{
			JobName: "contingency_retransmission",
			FailedDocuments: []event.RejectedDocSummary{
				{
					ControlNumber: "DTE-01-0001-000000001",
					ErrorCode:     "MH-REJECT-01",
					Description:   "DTE inválido: esquema JSON incorrecto",
					Observations:  []string{"El campo emisor.nit no cumple el formato requerido"},
				},
				{
					ControlNumber: "DTE-03-0001-000000002",
					ErrorCode:     "MH-REJECT-02",
					Description:   "DTE duplicado: código de generación ya procesado",
					Observations:  []string{},
				},
			},
			OccurredAtTime: now,
		}
	default:
		return event.ContingencyActivatedEvent{
			BranchID:        1,
			BranchAddress:   "San Salvador, San Salvador – Calle Principal #123",
			NIT:             "00000000000000",
			ClientName:      "Empresa de Prueba S.A. de C.V.",
			ContingencyType: "1",
			Reason:          "Evento de prueba enviado desde el endpoint /debug/notify-test",
			AffectedDocs:    1,
			OccurredAtTime:  now,
		}
	}
}
