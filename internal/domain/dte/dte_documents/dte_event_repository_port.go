package dte_documents

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
)

// DTEEventRepositoryPort persiste los eventos transmitidos a Hacienda (invalidacion, contingencia)
type DTEEventRepositoryPort interface {
	Create(ctx context.Context, event *dte.DTEEvent) error
	ListByRelatedDTE(ctx context.Context, branchID uint, relatedDTEID string) ([]dte.DTEEvent, error)
}
