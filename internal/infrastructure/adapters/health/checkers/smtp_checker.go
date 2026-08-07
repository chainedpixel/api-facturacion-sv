package checkers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/wneessen/go-mail"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type smtpChecker struct{}

func NewSMTPChecker() health.ComponentChecker {
	return &smtpChecker{}
}

func (c *smtpChecker) Name() string { return "smtp" }

func (c *smtpChecker) Check() models.Health {
	if config.SMTP == nil || config.SMTP.Host == "" {
		return models.Health{
			Status:  constants.StatusUp,
			Details: utils.TranslateHealthNotConfigured(c.Name()),
		}
	}

	port := 587
	if config.SMTP.Port != "" {
		if p, err := strconv.Atoi(config.SMTP.Port); err == nil && p > 0 {
			port = p
		}
	}

	opts := []mail.Option{mail.WithPort(port)}
	switch {
	case config.SMTP.TLS && port == 465:
		opts = append(opts, mail.WithSSL())
	case config.SMTP.TLS:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	default:
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}
	if config.SMTP.Username != "" || config.SMTP.Password != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(config.SMTP.Username),
			mail.WithPassword(config.SMTP.Password),
		)
	}

	client, err := mail.NewClient(config.SMTP.Host, opts...)
	if err != nil {
		return c.down(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.DialWithContext(ctx); err != nil {
		return c.down(err)
	}
	_ = client.Close()

	return models.Health{
		Status:  constants.StatusUp,
		Details: utils.TranslateHealthUp(c.Name()),
	}
}

func (c *smtpChecker) down(err error) models.Health {
	return models.Health{
		Status:  constants.StatusDown,
		Details: utils.TranslateMessage("health.down", classifySMTPError(err)),
	}
}

func classifySMTPError(err error) string {
	if err == nil {
		return "smtp"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "auth"),
		strings.Contains(msg, "credentials"),
		strings.Contains(msg, "535"),
		strings.Contains(msg, "5.7.8"):
		return "smtp_invalid_credentials"
	case strings.Contains(msg, "no such host"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "i/o"),
		strings.Contains(msg, "eof"),
		strings.Contains(msg, "unreachable"),
		strings.Contains(msg, "dial"):
		return "smtp_unreachable"
	default:
		return "smtp"
	}
}
