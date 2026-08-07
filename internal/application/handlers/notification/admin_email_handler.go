package notification

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/notification"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/notifier/email"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type renderer interface {
	Render(evt event.Event) (subject, html, plain string, err error)
}

type AdminEmailHandler struct {
	mailer   notification.Mailer
	cooldown email.Cooldown
	render   renderer
}

func NewAdminEmailHandler(m notification.Mailer, c email.Cooldown, r renderer) *AdminEmailHandler {
	return &AdminEmailHandler{mailer: m, cooldown: c, render: r}
}

func (h *AdminEmailHandler) Name() string { return "admin_email_handler" }

func (h *AdminEmailHandler) Handle(ctx context.Context, evt event.Event) error {
	if h.cooldown != nil {
		ok, err := h.cooldown.ShouldNotify(ctx, evt.Name(), evt.AggregateID())
		if err != nil {
			logs.Warn("cooldown check failed; proceeding to send", map[string]interface{}{
				"event": evt.Name(),
				"error": err.Error(),
			})
		} else if !ok {
			logs.Debug("admin email skipped: cooldown active", map[string]interface{}{
				"event":        evt.Name(),
				"aggregate_id": evt.AggregateID(),
			})
			return nil
		}
	}

	subject, html, plain, err := h.render.Render(evt)
	if err != nil {
		logs.Error("failed to render admin email", map[string]interface{}{
			"event": evt.Name(),
			"error": err.Error(),
		})
		return err
	}

	if err := h.mailer.SendAdminAlert(ctx, subject, html, plain); err != nil {
		logs.Error("failed to send admin email", map[string]interface{}{
			"event": evt.Name(),
			"error": err.Error(),
		})
		return err
	}

	return nil
}
