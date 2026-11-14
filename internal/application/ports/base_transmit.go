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

// TransmissionError carries the JWS that was transmitted when the transmission failed, so a rejected attempt can be
// kept as evidence. It unwraps to the original error.
type TransmissionError struct {
	Err            error
	SignedDocument string
}

// Error returns the message of the original error.
func (e *TransmissionError) Error() string {
	if e.Err == nil {
		return "transmission failed"
	}
	return e.Err.Error()
}

// Unwrap returns the original error.
func (e *TransmissionError) Unwrap() error { return e.Err }

type SignerManager interface {
	SignDTE(ctx context.Context, dte json.RawMessage, nit string) (string, error)
}
