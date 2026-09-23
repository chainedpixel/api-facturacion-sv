package response_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/invoice"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// ToMHInvoice converts an ElectronicInvoice to the structure required by the Ministry of Finance
func ToMHInvoice(doc interface{}) *structs.InvoiceDTEResponse {
	cast := doc.(*invoice_models.ElectronicInvoice)
	resumen := invoice.MapInvoiceResponseSummary(cast.InvoiceSummary)

	dte := &structs.InvoiceDTEResponse{
		Identificacion:  common.MapCommonResponseIdentification(cast.Identification),
		Emisor:          common.MapCommonResponseIssuer(cast.Issuer),
		Receptor:        invoice.MapInvoiceResponseReceiver(cast.Receiver),
		Resumen:         resumen,
		CuerpoDocumento: invoice.MapInvoiceResponseItem(cast.InvoiceItems),
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

	if cast.Appendix != nil {
		dte.Apendice = common.MapCommonResponseAppendix(cast.Appendix)
	}

	return dte
}
