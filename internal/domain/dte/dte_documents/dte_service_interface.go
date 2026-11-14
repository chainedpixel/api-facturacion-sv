package dte_documents

import (
	"context"
	"encoding/json"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
)

// DTEManager is an interface that defines the methods of a DTE manager.
type DTEManager interface {
	Create(ctx context.Context, document interface{}, transmission, status string, evidence *dte.TransmissionEvidence) error
	UpdateDTE(ctx context.Context, branchID uint, document dte.DTEDetails) error
	VerifyStatus(ctx context.Context, branchID uint, id string) (string, error)
	GetByGenerationCode(ctx context.Context, branchID uint, generationCode string) (*dte.DTEDocument, error)
	GenerateBalanceTransaction(ctx context.Context, branchID uint, transactionType, id, originalDTE string, document interface{}) error
	GenerateBalanceTransactionWithAmounts(ctx context.Context, branchID uint, transactionType, originalDTE, adjustmentDTE string, taxedSale, exemptSale, notSubjectSale float64) error
	ValidateForCreditNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
	ValidateForDebitNote(ctx context.Context, branchID uint, originalDTE string, document interface{}) error
	GetByGenerationCodeConsult(ctx context.Context, branchID uint, generationCode string) (*dte.DTEResponse, error)
	GetAllDTEs(ctx context.Context, filters *dte.DTEFilters) (*dte.DTEListResponse, error)
	RegisterEvent(ctx context.Context, event *dte.DTEEvent) error
	GetArchive(ctx context.Context, branchID uint, generationCode string) (json.RawMessage, error)
	GetEvents(ctx context.Context, branchID uint, generationCode string) ([]dte.EventArchiveResponse, error)
	GetActiveAdjustmentNotes(ctx context.Context, branchID uint, originalDTE string) ([]string, error)
}
