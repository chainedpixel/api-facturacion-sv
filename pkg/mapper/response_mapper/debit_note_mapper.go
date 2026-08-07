package response_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/debit_note"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func ToMHDebitNote(doc interface{}) *structs.DebitNoteDTEResponse {
	cast := doc.(*debit_note_models.DebitNoteModel)
	dte := &structs.DebitNoteDTEResponse{
		Identificacion:  common.MapCommonResponseIdentification(cast.Identification),
		Receptor:        common.MapCommonResponseReceiver(cast.Receiver),
		Emisor:          debit_note.MapDebitNoteIssuer(cast.Issuer),
		Resumen:         debit_note.MapDebitNoteResponseSummary(cast.DebitSummary),
		CuerpoDocumento: debit_note.MapDebitNoteResponseItem(cast.DebitItems),
		Extension:       debit_note.MapDebitNoteResponseExtension(cast.Extension),
	}

	dte.DocumentoRelacionado = common.MapCommonResponseRelatedDocuments(cast.GetRelatedDocuments())

	if cast.GetThirdPartySale() != nil {
		dte.VentaTercero = common.MapCommonResponseThirdPartySale(cast.GetThirdPartySale())
	}

	if cast.GetAppendix() != nil {
		dte.Apendice = common.MapCommonResponseAppendix(cast.GetAppendix())
	}

	return dte
}
