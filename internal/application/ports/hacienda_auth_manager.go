package ports

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
)

type HaciendaAuthManager interface {
	GetOrCreateHaciendaToken(ctx context.Context, systemToken string) (string, error)
	GetOrCreateHaciendaTokenWithCreds(ctx context.Context, systemToken string, creds models.HaciendaCredentials) (string, error)
}
