package remission_note_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"

type RemissionNoteModel struct {
	*models.DTEDocument
	RemissionItems []RemissionNoteItem
	Summary        *RemissionNoteSummary
}
