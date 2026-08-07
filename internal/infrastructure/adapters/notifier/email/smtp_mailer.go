package email

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wneessen/go-mail"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/notification"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type SMTPMailer struct {
	host       string
	port       int
	username   string
	password   string
	from       string
	tls        bool
	adminEmail string
}

func NewSMTPMailer() notification.Mailer {
	port := 587
	if config.SMTP != nil && config.SMTP.Port != "" {
		if p, err := strconv.Atoi(config.SMTP.Port); err == nil && p > 0 {
			port = p
		}
	}
	m := &SMTPMailer{port: port}
	if config.SMTP != nil {
		m.host = config.SMTP.Host
		m.username = config.SMTP.Username
		m.password = config.SMTP.Password
		m.from = config.SMTP.From
		m.tls = config.SMTP.TLS
	}
	if config.Server != nil {
		m.adminEmail = config.Server.AdminEmail
	}
	return m
}

func (m *SMTPMailer) SendAdminAlert(ctx context.Context, subject, htmlBody, plainBody string) error {
	if m.host == "" {
		logs.Warn("admin email skipped: SMTP_HOST not configured", map[string]interface{}{"subject": subject})
		return nil
	}
	if m.adminEmail == "" {
		logs.Warn("admin email skipped: ADMIN_EMAIL not configured", map[string]interface{}{"subject": subject})
		return nil
	}
	if m.from == "" {
		logs.Warn("admin email skipped: SMTP_FROM not configured", map[string]interface{}{"subject": subject})
		return nil
	}

	msg := mail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("set From: %w", err)
	}
	if err := msg.To(m.adminEmail); err != nil {
		return fmt.Errorf("set To: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, plainBody)
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody)

	opts := []mail.Option{mail.WithPort(m.port)}
	switch {
	case m.tls && m.port == 465:
		opts = append(opts, mail.WithSSL())
	case m.tls:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	default:
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}
	if m.username != "" || m.password != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(m.username),
			mail.WithPassword(m.password),
		)
	}

	client, err := mail.NewClient(m.host, opts...)
	if err != nil {
		return fmt.Errorf("build mail client: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	logs.Info("admin email sent", map[string]interface{}{
		"subject": subject,
		"to":      m.adminEmail,
	})
	return nil
}
