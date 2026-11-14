package invoice_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"

type InvoiceData struct {
	*models.InputDataCommon
	Items          []InvoiceItem
	InvoiceSummary *InvoiceSummary
}
