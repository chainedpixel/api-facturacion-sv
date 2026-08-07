package contingency

import (
	"context"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
)

// ContingencyRepositoryPort interface for the contingency repository (already exists)
type ContingencyRepositoryPort interface {
	Create(ctx context.Context, doc *dte.ContingencyDocument) error
	GetPending(ctx context.Context, limit int) ([]dte.ContingencyDocument, error)
	UpdateBatch(ctx context.Context, ids []string, observations []string, stamps map[string]string, batchID string, mhBatchID string, status string) error
	GetFirstContingencyTimestamp(ctx context.Context, branchID uint) (*time.Time, error)
}
