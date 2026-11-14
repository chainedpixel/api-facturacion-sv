package invalidation

import (
	"context"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// InvalidationManager is the interface that defines the methods that can be performed on document invalidation
type InvalidationManager interface {
	Validate(ctx context.Context, branchID uint, document *invalidation_models.InvalidationDocument) error
	ValidateStatus(ctx context.Context, branchID uint, req structs.CreateInvalidationRequest) error
	ValidateDependencies(ctx context.Context, branchID uint, doc *dte.DTEDocument) error
	InvalidateDocument(ctx context.Context, branchID uint, originalCode string) error
}
