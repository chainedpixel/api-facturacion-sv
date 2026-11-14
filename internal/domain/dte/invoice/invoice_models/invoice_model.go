package invoice_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"

type ElectronicInvoice struct {
	*models.DTEDocument `json:"*Models.DTEDocument"`
	InvoiceItems        []InvoiceItem  `json:"invoiceItems"`
	InvoiceSummary      InvoiceSummary `json:"invoiceSummary"`
}
