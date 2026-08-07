package notification

import "context"

type Mailer interface {
	SendAdminAlert(ctx context.Context, subject, htmlBody, plainBody string) error
}
