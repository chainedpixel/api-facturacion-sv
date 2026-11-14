package ports

import (
	"context"
	"encoding/json"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
)

type BaseTransmitter interface {
	RetryTransmission(ctx context.Context, document interface{}, token string, nit string) (*models.TransmitResult, error)
	CheckStatus(ctx context.Context, document interface{}, nit string, systemToken string) (*models.TransmitResult, error)
}

type SignerManager interface {
	SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
