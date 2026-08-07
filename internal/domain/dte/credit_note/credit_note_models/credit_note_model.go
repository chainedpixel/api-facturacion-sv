package credit_note_models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type CreditNoteModel struct {
	*models.DTEDocument
	CreditItems   []CreditNoteItem
	CreditSummary CreditNoteSummary
}
