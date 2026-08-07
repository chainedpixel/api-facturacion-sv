package ports

import (
	"context"
	"encoding/json"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
)

type BaseTransmitter interface {
	RetryTransmission(ctx context.Context, document interface{}, token string, nit string) (*models.TransmitResult, error)
	CheckStatus(ctx context.Context, document interface{}, nit string) (*models.TransmitResult, error)
}

type SignerManager interface {
	SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
