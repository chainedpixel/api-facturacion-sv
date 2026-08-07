package response_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/ccf"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func ToMHCreditFiscalInvoice(doc interface{}) *structs.CCFDTEResponse {

	cast := doc.(*ccf_models.CreditFiscalDocument)
	dte := &structs.CCFDTEResponse{
		Identificacion:  common.MapCommonResponseIdentification(cast.Identification),
		Emisor:          common.MapCommonResponseIssuer(cast.Issuer),
		Receptor:        common.MapCommonResponseReceiver(cast.Receiver),
		Resumen:         ccf.MapCCFResponseSummary(cast.CreditSummary),
		CuerpoDocumento: ccf.MapCCFResponseItem(cast.CreditItems),
		Extension:       common.MapCommonResponseExtension(cast.Extension),
	}

	if len(cast.GetRelatedDocuments()) > 0 {
		dte.DocumentoRelacionado = common.MapCommonResponseRelatedDocuments(cast.GetRelatedDocuments())
	}

	if len(cast.GetOtherDocuments()) > 0 {
		dte.OtrosDocumentos = common.MapCommonResponseOtherDocuments(cast.GetOtherDocuments())
	}

	if cast.GetThirdPartySale() != nil {
		dte.VentaTercero = common.MapCommonResponseThirdPartySale(cast.GetThirdPartySale())
	}

	if cast.GetAppendix() != nil {
		dte.Apendice = common.MapCommonResponseAppendix(cast.GetAppendix())
	}

	return dte
}
