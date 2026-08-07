package debit_note_models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type DebitNoteInput struct {
	*models.InputDataCommon
	Items        []DebitNoteItem
	DebitSummary *DebitNoteSummary
}
