package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/shopspring/decimal"
)

type PaymentTotalStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Validates the payment total rules of a DTE document
func (s *PaymentTotalStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetSummary() == nil {
		return nil
	}

	if s.Document.GetIdentification().GetDTEType() == constants.NotaCreditoElectronica ||
		s.Document.GetIdentification().GetDTEType() == constants.NotaDebitoElectronica ||
		s.Document.GetIdentification().GetDTEType() == constants.NotaRemisionElectronica {
		return nil
	}

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
	for _, payment := range s.Document.GetSummary().GetPaymentTypes() {
		amount := decimal.NewFromFloat(payment.GetAmount())
		paymentsTotal = paymentsTotal.Add(amount)
	}

	operationTotal := decimal.NewFromFloat(s.Document.GetSummary().GetTotalToPay())

	diff := operationTotal.Sub(paymentsTotal)
	if diff.Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		return dte_errors.NewDTEErrorSimple("InvalidPaymentTotal",
			paymentsTotal.InexactFloat64(), operationTotal.InexactFloat64())
	}

	return nil
}

// dontHaveCashPayment Checks whether any cash payment type exists
func (s *PaymentTotalStrategy) dontHaveCashPayment() bool {
	for _, payment := range s.Document.GetSummary().GetPaymentTypes() {
		if payment.GetCode() == constants.BilletesMonedas {
			return false
		}
	}

	return true
}
