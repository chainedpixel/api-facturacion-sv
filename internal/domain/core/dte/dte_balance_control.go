package dte

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
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
