package credit_note_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"

type CreditNoteModel struct {
	*models.DTEDocument
	CreditItems   []CreditNoteItem
	CreditSummary CreditNoteSummary
}
