package strategy

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/shopspring/decimal"
)

type PaymentTotalStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Validates the payment rules of a DTE document according to its operation condition; remission notes are
// skipped and credit and debit notes only require the cash condition when the total to pay is 0.00
func (s *PaymentTotalStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetSummary() == nil {
		return nil
	}

	dteType := s.Document.GetIdentification().GetDTEType()
	summary := s.Document.GetSummary()
	payments := summary.GetPaymentTypes()
	nothingDue := decimal.NewFromFloat(summary.GetTotalToPay()).IsZero()

	switch dteType {
	case constants.NotaRemisionElectronica:
		return nil
	case constants.NotaCreditoElectronica, constants.NotaDebitoElectronica:
		if nothingDue {
			return validateCashCondition(summary.GetOperationCondition())
		}
		return nil
	}

	if nothingDue {
		if dteType == constants.FacturaSujetoExcluidoElectronica {
			if err := validateCashCondition(summary.GetOperationCondition()); err != nil {
				return err
			}
		} else if err := ValidateNoPaymentDue(summary.GetOperationCondition(), payments); err != nil {
			return err
		}
	}

	if summary.GetOperationCondition() == constants.Credit && len(payments) > 0 {
		return validateCreditPayments(payments)
	}

	for _, payment := range payments {
		if payment.GetCode() == "" {
			return dte_errors.NewDTEErrorSimple("PaymentCodeRequired", summary.GetOperationCondition())
		}

		if s.dontHaveCashPayment() && summary.GetOperationCondition() == constants.Cash {
			if payment.GetTerm() != nil || payment.GetPeriod() != nil {
				return dte_errors.NewDTEErrorSimple("InvalidPaymentTermsOF")
			}
		}
	}

	paymentsTotal := decimal.Zero
	for _, payment := range payments {
		paymentsTotal = paymentsTotal.Add(decimal.NewFromFloat(payment.GetAmount()))
	}

	operationTotal := decimal.NewFromFloat(summary.GetTotalToPay())

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

// validateCreditPayments Requires every payment of a credit operation to carry a null code, a 0.00 amount,
// a null reference, a term and a period greater than zero
func validateCreditPayments(payments []interfaces.PaymentType) *dte_errors.DTEError {
	for _, payment := range payments {
		if payment.GetCode() != "" {
			return dte_errors.NewDTEErrorSimple("CreditPaymentCodeNotAllowed", payment.GetCode())
		}

		if !decimal.NewFromFloat(payment.GetAmount()).IsZero() {
			return dte_errors.NewDTEErrorSimple("InvalidCreditPaymentAmount", payment.GetAmount())
		}

		if payment.GetReference() != "" {
			return dte_errors.NewDTEErrorSimple("CreditPaymentReferenceNotAllowed", payment.GetReference())
		}

		if payment.GetTerm() == nil {
			return dte_errors.NewDTEErrorSimple("CreditPaymentTermRequired")
		}

		if payment.GetPeriod() == nil || *payment.GetPeriod() <= 0 {
			return dte_errors.NewDTEErrorSimple("CreditPaymentPeriodRequired")
		}
	}

	return nil
}

// ValidateNoPaymentDue Requires a cash operation condition and at least one payment, all with code 99, when nothing
// is due in an FE, CCF or FEXE: the total to pay is 0.00 or the document carries a balance in favor
func ValidateNoPaymentDue(condition int, payments []interfaces.PaymentType) *dte_errors.DTEError {
	if err := validateCashCondition(condition); err != nil {
		return err
	}

	if len(payments) == 0 {
		return dte_errors.NewDTEErrorSimple("NoPaymentDueRequiresCode99")
	}

	for _, payment := range payments {
		if payment.GetCode() != constants.NoAplica {
			return dte_errors.NewDTEErrorSimple("NoPaymentDueRequiresCode99")
		}
	}

	return nil
}

// ValidateBalanceInFavorPayments applies ValidateNoPaymentDue to a document that carries a balance in favor; a zero
// total to pay is already covered by PaymentTotalStrategy and is skipped to avoid reporting the same rule twice
func ValidateBalanceInFavorPayments(summary interfaces.Summary, balanceInFavor float64) *dte_errors.DTEError {
	if summary == nil || decimal.NewFromFloat(balanceInFavor).IsZero() || decimal.NewFromFloat(summary.GetTotalToPay()).IsZero() {
		return nil
	}

	return ValidateNoPaymentDue(summary.GetOperationCondition(), summary.GetPaymentTypes())
}

// validateCashCondition Requires the cash operation condition, used when nothing is due on the document
func validateCashCondition(condition int) *dte_errors.DTEError {
	if condition != constants.Cash {
		return dte_errors.NewDTEErrorSimple("NoPaymentDueRequiresCashCondition", condition)
	}

	return nil
}
