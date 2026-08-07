package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// MapCommonRequestPaymentsType maps an array of payments to a payments model -> Source: Request
func MapCommonRequestPaymentsType(payments []structs.PaymentRequest) ([]interfaces.PaymentType, error) {
	result := make([]interfaces.PaymentType, len(payments))

	for i, payment := range payments {
		var term *financial.PaymentTerm

		if payment.Code == "" || payment.Amount == 0 {
			return nil, shared_error.NewFormattedGeneralServiceError(
				"CommonMapper",
				"MapCommonRequestPaymentsType",
				"InvalidPaymentTypeInfo",
			)
		}

		paymentCode, err := financial.NewPaymentType(payment.Code)
		if err != nil {
			return nil, err
		}

		amount, err := financial.NewAmount(payment.Amount)
		if err != nil {
			return nil, err
		}

		if payment.Term != nil {
			term, err = financial.NewPaymentTerm(*payment.Term)
			if err != nil {
				return nil, err
			}
		}

		if payment.Reference == nil {
			payment.Reference = new(string)
		}

		result[i] = &models.PaymentType{
			Code:      *paymentCode,
			Amount:    *amount,
			Reference: *payment.Reference,
			Period:    payment.Period,
			Term:      term,
		}
	}

	return result, nil
}
