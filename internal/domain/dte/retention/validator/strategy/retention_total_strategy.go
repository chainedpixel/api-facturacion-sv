package strategy

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/retention/retention_models"
	"github.com/shopspring/decimal"
)

type RetentionTotalStrategy struct {
	Document *retention_models.RetentionModel
}

func (s *RetentionTotalStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return nil
	}

	validations := []func() *dte_errors.DTEError{
		s.validateTotalAmounts,
		s.validateTotalIVA,
	}

	for _, validate := range validations {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

// validateTotalAmounts validates the retention totals of the document
func (s *RetentionTotalStrategy) validateTotalAmounts() *dte_errors.DTEError {
	expectedTotalSubjectRetention := s.Document.RetentionSummary.TotalSubjectRetention.GetValueAsDecimal()
	actualIvaRetention := s.Document.RetentionSummary.TotalIVARetention.GetValueAsDecimal()

	actualTotalSubjectRetention, expectedIvaRetention := s.Document.GetTotalByItems()
	if !s.compareTotalsWithTolerance(expectedTotalSubjectRetention, actualTotalSubjectRetention, 0.01) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalSubjectRetention",
			actualTotalSubjectRetention.InexactFloat64(),
			expectedTotalSubjectRetention.InexactFloat64())
	}

	if !s.compareTotalsWithTolerance(expectedIvaRetention, actualIvaRetention, 0.01) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalIVARetention",
			actualIvaRetention.InexactFloat64(),
			expectedIvaRetention.InexactFloat64())
	}

	return nil
}

// compareTotalsWithTolerance compares two totals with a specified tolerance
func (s *RetentionTotalStrategy) compareTotalsWithTolerance(expected, actual decimal.Decimal, tolerance float64) bool {
	diff := expected.Sub(actual).Abs()
	return diff.LessThanOrEqual(decimal.NewFromFloat(tolerance))
}

// validateTotalIVA valida el IVA 13% del CRE (Anexo IV, campo 157) con retention_models.ExpectedTotalIVA
func (s *RetentionTotalStrategy) validateTotalIVA() *dte_errors.DTEError {
	code := ""
	allElectronic := len(s.Document.RetentionItems) > 0
	for _, item := range s.Document.RetentionItems {
		code = item.ReceptionCodeMH.GetValue()
		if item.DocumentType.GetValue() != constants.ElectronicDocument {
			allElectronic = false
		}
	}
	expected := decimal.NewFromFloat(retention_models.ExpectedTotalIVA(
		s.Document.RetentionSummary.TotalSubjectRetention.GetValue(), code, allElectronic))
	actual := s.Document.RetentionSummary.TotalIVA.GetValueAsDecimal()

	if !s.compareTotalsWithTolerance(expected, actual, 0.01) {
		return dte_errors.NewDTEErrorSimple("InvalidTotalIVA", actual.InexactFloat64(), expected.InexactFloat64())
	}
	return nil
}
