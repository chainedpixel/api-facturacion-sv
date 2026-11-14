package dte

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/shopspring/decimal"
)

type BalanceControl struct {
	ID                        uint      `json:"-"`
	BranchID                  uint      `json:"-"`
	OriginalDTEID             string    `json:"original_dte_id"`
	OriginalTaxedAmount       float64   `json:"original_taxed_amount"`
	OriginalExemptAmount      float64   `json:"original_exempt_amount"`
	OriginalNotSubjectAmount  float64   `json:"original_not_subject_amount"`
	RemainingTaxedAmount      float64   `json:"remaining_taxed_amount"`
	RemainingExemptAmount     float64   `json:"remaining_exempt_amount"`
	RemainingNotSubjectAmount float64   `json:"remaining_not_subject_amount"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`

	OriginalDTE  *DTEDetails          `json:"original_dte,omitempty"`
	Branch       *user.BranchOffice   `json:"branch,omitempty"`
	Transactions []BalanceTransaction `json:"transactions,omitempty"`
}

// DebitNotesNotSubjectAmount returns the non-subject amount of the debit notes still in force on the document: the
// debit note transactions minus the transactions recorded when a debit note is invalidated.
func (b *BalanceControl) DebitNotesNotSubjectAmount() float64 {
	total := decimal.Zero
	for _, transaction := range b.Transactions {
		switch transaction.TransactionType {
		case constants.NotaDebitoElectronica:
			total = total.Add(decimal.NewFromFloat(transaction.NotSubjectAmount))
		case constants.DebitNoteInvalid:
			total = total.Sub(decimal.NewFromFloat(transaction.NotSubjectAmount))
		}
	}
	return total.Round(2).InexactFloat64()
}
