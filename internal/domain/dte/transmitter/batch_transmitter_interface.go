package transmitter

import (
	"context"

	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
)

// BatchTransmitterPort interface for batch transmission to Hacienda
type BatchTransmitterPort interface {
	TransmitBatch(ctx context.Context, systemNIT string, dteType string, documents []string, token string, credentials authModels.HaciendaCredentials) (*models.BatchResponse, string, error)
	VerifyContingencyBatchStatus(ctx context.Context, batchID string, mhBatchID string, token string, branchID uint, docsMap map[string]dte.ContingencyDocument) error
	GetDTEVersion(dteType string) int
}
