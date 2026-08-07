package invoice_models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type InvoiceData struct {
	*models.InputDataCommon
	Items          []InvoiceItem
	InvoiceSummary *InvoiceSummary
}
