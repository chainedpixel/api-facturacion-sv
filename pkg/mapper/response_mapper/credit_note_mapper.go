package response_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/credit_note"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func ToMHCreditNote(doc interface{}) *structs.CreditNoteDTEResponse {

	cast := doc.(*credit_note_models.CreditNoteModel)
	dte := &structs.CreditNoteDTEResponse{
		Identificacion:  &structs.DTEAdjustmentIdentification{DTEIdentification: common.MapCommonResponseIdentification(cast.Identification)},
		Receptor:        common.MapCommonResponseDocumentReceiver(cast.Receiver),
		Emisor:          credit_note.MapCreditNoteIssuer(cast.Issuer),
		Resumen:         credit_note.MapCreditNoteResponseSummary(cast.CreditSummary),
		CuerpoDocumento: credit_note.MapCreditNoteResponseItem(cast.CreditItems),
	}

	dte.DocumentoRelacionado = common.MapCommonResponseRelatedDocuments(cast.GetRelatedDocuments())

	for i := range dte.CuerpoDocumento {
		dte.CuerpoDocumento[i].IvaPerci = common.AdjustmentItemIVA(dte.CuerpoDocumento[i].VentaGravada, dte.Resumen.IvaPerci)
		dte.CuerpoDocumento[i].IvaRete = common.AdjustmentItemIVA(dte.CuerpoDocumento[i].VentaGravada, dte.Resumen.IvaRete)
	}

	if !common.AdjustsRetentionVoucher(cast.GetRelatedDocuments()) {
		dte.Resumen.TotalIva = 0
		for i := range dte.CuerpoDocumento {
			dte.CuerpoDocumento[i].TotalIva = 0
		}
	}

	if cast.GetThirdPartySale() != nil {
		dte.VentaTercero = common.MapCommonResponseThirdPartySale(cast.GetThirdPartySale())
	}

	if cast.GetAppendix() != nil {
		dte.Apendice = common.MapCommonResponseAppendix(cast.GetAppendix())
	}

	return dte
}
