package dte_documents

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
)

// DTEManager is an interface that defines the methods of a DTE manager.
type DTEManager interface {
	Create(context.Context, interface{}, string, string, *string) error
	UpdateDTE(ctx context.Context, branchID uint, document dte.DTEDetails) error
	VerifyStatus(ctx context.Context, branchID uint, id string) (string, error)
	GetByGenerationCode(ctx context.Context, branchID uint, generationCode string) (*dte.DTEDocument, error)
	GenerateBalanceTransaction(ctx context.Context, branchID uint, transactionType, id, originalDTE string, document interface{}) error
	GenerateBalanceTransactionWithAmounts(ctx context.Context, branchID uint, transactionType, originalDTE, adjustmentDTE string, taxedSale, exemptSale, notSubjectSale float64) error
	ValidateForCreditNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
	ValidateForDebitNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
	GetByGenerationCodeConsult(ctx context.Context, branchID uint, generationCode string) (*dte.DTEResponse, error)
	GetAllDTEs(ctx context.Context, filters *dte.DTEFilters) (*dte.DTEListResponse, error)
}
