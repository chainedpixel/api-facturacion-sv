package dte_documents

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
)

// DTERepositoryPort is an interface that defines the methods of a DTE repository.
type DTERepositoryPort interface {
	Create(ctx context.Context, document interface{}, transmission, status string, receptionStamp *string) error
	Update(ctx context.Context, branchID uint, document dte.DTEDetails) error
	GetByGenerationCode(ctx context.Context, branchID uint, id string) (*dte.DTEDocument, error)
	GetDTEBalanceControl(ctx context.Context, branchID uint, id string) (*dte.BalanceControl, error)
	GenerateBalanceTransaction(ctx context.Context, branchID uint, originalDTE string, transaction *dte.BalanceTransaction) error
	VerifyStatus(ctx context.Context, branchID uint, id string) (string, error)
	GetTotalCount(ctx context.Context, filters *dte.DTEFilters) (int64, error)
	GetSummaryStats(ctx context.Context, filters *dte.DTEFilters) (*dte.ListSummary, error)
	GetPagedDocuments(ctx context.Context, filters *dte.DTEFilters) ([]dte.DTEModelResponse, error)
}
