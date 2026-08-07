package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
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
