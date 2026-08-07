package credit_note_models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type CreditNoteInput struct {
	*models.InputDataCommon
	Items         []CreditNoteItem
	CreditSummary *CreditNoteSummary
}
