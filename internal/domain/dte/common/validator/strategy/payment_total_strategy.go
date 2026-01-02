package strategy

import (
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/shopspring/decimal"
)

type PaymentTotalStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Válida las reglas de total de pagos de un documento DTE
func (s *PaymentTotalStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetSummary() == nil {
		return nil
	}

	if s.Document.GetIdentification().GetDTEType() == constants.NotaCreditoElectronica {
		return nil
	}

	// Validar términos de pago para crédito
	for _, payment := range s.Document.GetSummary().GetPaymentTypes() {
		if s.Document.GetSummary().GetOperationCondition() == constants.Credit && s.dontHaveCashPayment() {
			if payment.GetCode() == constants.BilletesMonedas {
				return dte_errors.NewDTEErrorSimple("InvalidPaymentTypeOP2")
			}

			if payment.GetTerm() == nil || payment.GetPeriod() == nil {
				return dte_errors.NewDTEErrorSimple("InvalidPaymentTerms")
			}
		}

		if s.dontHaveCashPayment() && s.Document.GetSummary().GetOperationCondition() == constants.Cash {
			if payment.GetTerm() != nil || payment.GetPeriod() != nil {
				return dte_errors.NewDTEErrorSimple("InvalidPaymentTermsOF")
			}
		}
	}

	paymentsTotal := decimal.Zero
	// Sumar todos los pagos
	for _, payment := range s.Document.GetSummary().GetPaymentTypes() {
		amount := decimal.NewFromFloat(payment.GetAmount())
		paymentsTotal = paymentsTotal.Add(amount)
	}

	operationTotal := decimal.NewFromFloat(s.Document.GetSummary().GetTotalToPay())

	// Validar que el total de pagos sea igual al total de operaciones
	diff := operationTotal.Sub(paymentsTotal)
	if diff.Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		return dte_errors.NewDTEErrorSimple("InvalidPaymentTotal",
			paymentsTotal.InexactFloat64(), operationTotal.InexactFloat64())
	}

	return nil
}

// dontHaveCashPayment Verifica si existe algún tipo de pago en efectivo
func (s *PaymentTotalStrategy) dontHaveCashPayment() bool {
	for _, payment := range s.Document.GetSummary().GetPaymentTypes() {
		if payment.GetCode() == constants.BilletesMonedas {
			return false
		}
	}

	return true
}
