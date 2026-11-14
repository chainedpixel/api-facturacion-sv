package debit_note_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"

type DebitNoteModel struct {
	*models.DTEDocument
	DebitItems   []DebitNoteItem
	DebitSummary DebitNoteSummary
}
