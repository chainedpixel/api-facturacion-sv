package common

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// MapCommonRequestPaymentsType maps an array of payments to a payments model; an empty code is kept empty
// so the domain rules decide whether the operation condition allows it -> Source: Request
func MapCommonRequestPaymentsType(payments []structs.PaymentRequest) ([]interfaces.PaymentType, error) {
	result := make([]interfaces.PaymentType, len(payments))

	for i, payment := range payments {
		var term *financial.PaymentTerm

		paymentCode := &financial.PaymentType{}
		if payment.Code != "" {
			validCode, err := financial.NewPaymentType(payment.Code)
			if err != nil {
				return nil, err
			}
			paymentCode = validCode
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
