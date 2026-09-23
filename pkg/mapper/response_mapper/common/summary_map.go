package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseSummary maps an invoice summary to an invoice summary model -> Source: Response
func MapCommonResponseSummary(summary interfaces.Summary) *structs.DTESummary {
	result := &structs.DTESummary{
		TotalNoSuj:          summary.GetTotalNonSubject(),
		TotalExenta:         summary.GetTotalExempt(),
		TotalGravada:        summary.GetTotalTaxed(),
		SubTotalVentas:      summary.GetSubtotalSales(),
		DescuNoSuj:          summary.GetNonSubjectDiscount(),
		DescuExenta:         summary.GetExemptDiscount(),
		PorcentajeDescuento: summary.GetDiscountPercentage(),
		TotalDescu:          summary.GetTotalDiscount(),
		SubTotal:            summary.GetSubTotal(),
		MontoTotalOperacion: summary.GetTotalOperation(),
		TotalNoGravado:      summary.GetTotalNotTaxed(),
		TotalPagar:          summary.GetTotalToPay(),
		TotalLetras:         summary.GetTotalInWords(),
		CondicionOperacion:  summary.GetOperationCondition(),
		Tributos:            MapTaxes(summary.GetTotalTaxes()),
		Observaciones:       summary.GetObservations(),
	}

	if len(summary.GetPaymentTypes()) > 0 {
		result.Pagos = MapCommonResponsePayments(summary.GetPaymentTypes())
	}

	return result
}
