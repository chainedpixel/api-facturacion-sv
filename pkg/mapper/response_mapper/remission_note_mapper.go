package response_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/remission_note"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// ToMHRemissionNote converts a Remission Note domain model to the format required by the Ministry of Finance
func ToMHRemissionNote(domain interface{}) interface{} {
	remissionNoteModel, ok := domain.(*remission_note_models.RemissionNoteModel)
	if !ok {
		return nil
	}

	return &structs.MHRemissionNote{
		Identification:   remission_note.MapRemissionNoteIdentification(remissionNoteModel),
		RelatedDocuments: remission_note.MapRemissionNoteRelatedDocuments(remissionNoteModel),
		Issuer:           *remission_note.MapRemissionNoteIssuer(remissionNoteModel),
		Receiver:         remission_note.MapRemissionNoteReceiver(remissionNoteModel),
		ThirdPartySale:   remission_note.MapRemissionNoteThirdPartySale(remissionNoteModel),
		DocumentBody:     remission_note.MapRemissionNoteItems(remissionNoteModel),
		Summary:          remission_note.MapRemissionNoteSummary(remissionNoteModel.Summary),
		Appendix:         remission_note.MapRemissionNoteAppendix(remissionNoteModel),
	}
}
