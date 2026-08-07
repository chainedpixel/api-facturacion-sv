package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// MapCommonResponsePayments maps payments to a payments model -> Source: Response
func MapCommonResponsePayments(payments []interfaces.PaymentType) []structs.DTEPayment {
	result := make([]structs.DTEPayment, len(payments))
	for i, payment := range payments {

		result[i] = structs.DTEPayment{
			Codigo:    payment.GetCode(),
			MontoPago: payment.GetAmount(),
		}

		if reference := payment.GetReference(); reference != "" {
			result[i].Referencia = utils.ToStringPointer(reference)
		}

		if term := payment.GetTerm(); term != nil {
			result[i].Plazo = utils.ToStringPointer(*term)
		}
		if period := payment.GetPeriod(); period != nil {
			result[i].Periodo = utils.ToIntPointer(*period)
		}
	}
	return result
}
