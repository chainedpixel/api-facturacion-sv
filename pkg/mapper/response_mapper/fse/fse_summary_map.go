package fse

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapFSEResponseSummary(summary fse_models.FSESummary) structs.FSESummaryResponse {
	totalDescu := summary.TotalDiscount.GetValue()

	var pagos *[]structs.DTEPayment
	if len(summary.PaymentTypes) > 0 {
		mappedPayments := common.MapCommonResponsePayments(summary.PaymentTypes)
		pagos = &mappedPayments
	}

	var observaciones *string
	if summary.GetObservations() != nil && *summary.GetObservations() != "" {
		observaciones = summary.GetObservations()
	}

	return structs.FSESummaryResponse{
		TotalCompra:        summary.TotalPurchase.GetValue(),
		Descu:              summary.GetNonSubjectDiscount(),
		TotalDescu:         totalDescu,
		SubTotal:           summary.SubTotal.GetValue(),
		IvaRete1:           summary.IVARetention.GetValue(),
		ReteRenta:          summary.IncomeRetention.GetValue(),
		TotalPagar:         summary.TotalToPay.GetValue(),
		TotalLetras:        summary.TotalInWords,
		CondicionOperacion: summary.OperationCondition.GetValue(),
		Pagos:              pagos,
		Observaciones:      observaciones,
	}
}
