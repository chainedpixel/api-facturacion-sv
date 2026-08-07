package dte

import (
	"context"
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

// DTEOperations defines specific operations for each DTE type
type DTEOperations struct{}

type AdditionalOperationsFunc func(ctx context.Context, result interface{}, branchID uint, mhModel interface{}) error

// NewDTEOperations creates a new instance of DTEOperations
func NewDTEOperations() *DTEOperations {
	return &DTEOperations{}
}

// GetCreditNoteOperations returns the additional operations for credit notes
func (o *DTEOperations) GetCreditNoteOperations(dteService dte_documents.DTEManager) AdditionalOperationsFunc {
	return func(ctx context.Context, result interface{}, branchID uint, mhModel interface{}) error {
		creditNote, ok := result.(*credit_note_models.CreditNoteModel)
		if !ok {
			return fmt.Errorf("invalid result type")
		}

		for _, relatedDoc := range creditNote.GetRelatedDocuments() {
			if relatedDoc.GetGenerationType() == constants.ElectronicDocument {
				err := dteService.GenerateBalanceTransaction(
					ctx,
					branchID,
					constants.NotaCreditoElectronica,
					relatedDoc.GetDocumentNumber(),
					creditNote.GetIdentification().GetGenerationCode(),
					mhModel,
				)
				if err != nil {
					logs.Warn("Failed to generate balance transaction", map[string]interface{}{"error": err.Error()})
					return err
				}
			}
		}
		return nil
	}
}

// GetDebitNoteOperations returns the additional operations for debit notes
func (o *DTEOperations) GetDebitNoteOperations(dteService dte_documents.DTEManager) AdditionalOperationsFunc {
	return func(ctx context.Context, result interface{}, branchID uint, mhModel interface{}) error {
		debitNote, ok := result.(*debit_note_models.DebitNoteModel)
		if !ok {
			return fmt.Errorf("invalid result type")
		}

		for _, relatedDoc := range debitNote.GetRelatedDocuments() {
			if relatedDoc.GetGenerationType() == constants.ElectronicDocument {
				err := dteService.GenerateBalanceTransaction(
					ctx,
					branchID,
					constants.NotaDebitoElectronica,
					relatedDoc.GetDocumentNumber(),
					debitNote.GetIdentification().GetGenerationCode(),
					mhModel,
				)
				if err != nil {
					logs.Warn("Failed to generate balance transaction", map[string]interface{}{"error": err.Error()})
					return err
				}
			}
		}
		return nil
	}
}

// GetNoOperation returns a no-op function for DTEs with no additional operations
func (o *DTEOperations) GetNoOperation() AdditionalOperationsFunc {
	return func(ctx context.Context, result interface{}, branchID uint, mhModel interface{}) error {
		return nil
	}
}
