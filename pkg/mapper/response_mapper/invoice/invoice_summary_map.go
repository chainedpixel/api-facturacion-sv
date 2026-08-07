package invoice

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapInvoiceResponseSummary maps an invoice summary to an invoice summary model -> Source: Response
func MapInvoiceResponseSummary(summary invoice_models.InvoiceSummary) *structs.InvoiceSummary {
	result := MapInvoiceSummary(summary)
	result.DescuGravada = summary.TaxedDiscount.GetValue()
	result.IvaRete1 = summary.IVARetention.GetValue()
	result.TotalIva = summary.TotalIva.GetValue()
	result.SaldoFavor = summary.BalanceInFavor.GetValue()
	result.ReteRenta = summary.IncomeRetention.GetValue()

	return result
}

func MapInvoiceSummary(summary interfaces.Summary) *structs.InvoiceSummary {
	result := &structs.InvoiceSummary{
		TotalNoSuj:          summary.GetTotalNonSubject(),
		TotalExenta:         summary.GetTotalExempt(),
		TotalGravada:        summary.GetTotalTaxed(),
		SubTotalVentas:      summary.GetSubtotalSales(),
		DescuNoSuj:          summary.GetNonSubjectDiscount(),
		DescuExenta:         summary.GetExemptDiscount(),
		DescuGravada:        summary.GetExemptDiscount(),
		PorcentajeDescuento: summary.GetDiscountPercentage(),
		TotalDescu:          summary.GetTotalDiscount(),
		SubTotal:            summary.GetSubTotal(),
		MontoTotalOperacion: summary.GetTotalOperation(),
		TotalNoGravado:      summary.GetTotalNotTaxed(),
		TotalPagar:          summary.GetTotalToPay(),
		TotalLetras:         summary.GetTotalInWords(),
		CondicionOperacion:  summary.GetOperationCondition(),
		Tributos:            common.MapTaxes(summary.GetTotalTaxes()),
	}

	if len(summary.GetPaymentTypes()) > 0 {
		result.Pagos = common.MapCommonResponsePayments(summary.GetPaymentTypes())
	}

	return result
}
