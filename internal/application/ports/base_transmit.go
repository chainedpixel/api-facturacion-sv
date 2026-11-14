package ports

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
)

type BaseTransmitter interface {
	RetryTransmission(ctx context.Context, document interface{}, token string, nit string) (*models.TransmitResult, error)
	CheckStatus(ctx context.Context, document interface{}, nit string, systemToken string) (*models.TransmitResult, error)
}

// ErrSigning identifies a failure of the signing service. A document that cannot be signed is never sent to
// contingency, because a contingency document must be delivered already signed.
var ErrSigning = errors.New("signing service failure")

type SignerManager interface {
	SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
