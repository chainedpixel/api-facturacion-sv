package strategy

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/shopspring/decimal"
)

type RetentionItemStrategy struct {
	Document *retention_models.RetentionModel
}

func (s *RetentionItemStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return nil
	}

	validations := []func() *dte_errors.DTEError{
		s.ValidateAmountAndCode,
		s.ValidateDateRange,
	}

	for _, validate := range validations {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

func (s *RetentionItemStrategy) ValidateAmountAndCode() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.RetentionItems) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredField", "RetentionItems")
	}

	for _, item := range s.Document.RetentionItems {
		expectedIvaRetention := item.RetentionAmount.GetValueAsDecimal().Mul(constants.GetRetentionAmount[item.ReceptionCodeMH.GetValue()])
		actualIvaRetention := item.RetentionIVA.GetValueAsDecimal()

		if !s.compareTotalsWithTolerance(expectedIvaRetention, actualIvaRetention, 0.01) {
			logs.Info("Evaluating IVA Retention", map[string]interface{}{
				"item_number": item.Number.GetValue(),
				"expected":    expectedIvaRetention.InexactFloat64(),
				"actual":      actualIvaRetention.InexactFloat64(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidRetentionIVA",
				item.Number.GetValue(),
				actualIvaRetention.InexactFloat64(),
				expectedIvaRetention.InexactFloat64())
		}
	}

	return nil
}

// ValidateDateRange validates that the dates of the physical documents are within the allowed period
func (s *RetentionItemStrategy) ValidateDateRange() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.RetentionItems) == 0 {
		return nil
	}

	creDate := s.Document.Identification.GetEmissionDate()
	for _, item := range s.Document.RetentionItems {

		docDate := item.EmissionDate.GetValue()
		if !isWithinAllowedPeriod(docDate, creDate) {
			logs.Info("Document date out of allowed range", map[string]interface{}{
				"item_number":    item.Number.GetValue(),
				"document_date":  item.EmissionDate.GetValue(),
				"retention_date": s.Document.Identification.GetEmissionDate(),
			})
			return dte_errors.NewDTEErrorSimple("DateOutOfAllowedRange",
				item.Number.GetValue(),
				item.DocumentNumber.GetValue())
		}
	}

	return nil
}

// isWithinAllowedPeriod checks if the CRE date is within the allowed period
func isWithinAllowedPeriod(documentDate, retentionDate time.Time) bool {
	docDate := time.Date(documentDate.Year(), documentDate.Month(), documentDate.Day(), 0, 0, 0, 0, time.UTC)
	retDate := time.Date(retentionDate.Year(), retentionDate.Month(), retentionDate.Day(), 0, 0, 0, 0, time.UTC)

	sameMonth := docDate.Year() == retDate.Year() && docDate.Month() == retDate.Month()

	var previousMonth bool
	if docDate.Month() == time.December {
		previousMonth = docDate.Year()+1 == retDate.Year() && retDate.Month() == time.January
	} else {
		previousMonth = docDate.Year() == retDate.Year() && docDate.Month()+1 == retDate.Month()
	}

	if !sameMonth && !previousMonth {
		return false
	}

	if sameMonth {
		return true
	}

	firstDayNextMonth := time.Date(docDate.Year(), docDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	lastAllowedDay := addBusinessDays(firstDayNextMonth, 10)

	return retDate.Before(lastAllowedDay) || retDate.Equal(lastAllowedDay)
}

// addBusinessDays adds business days to a date (excluding weekends)
func addBusinessDays(date time.Time, days int) time.Time {
	result := date
	added := 0

	for added < days {
		result = result.AddDate(0, 0, 1)

		if result.Weekday() != time.Saturday && result.Weekday() != time.Sunday {
			added++
		}
	}

	return result
}

// addMonths adds months to a date
func addMonths(date time.Time, months int) time.Time {
	return date.AddDate(0, months, 0)
}

// compareTotalsWithTolerance compares two totals with a specified tolerance
func (s *RetentionItemStrategy) compareTotalsWithTolerance(expected, actual decimal.Decimal, tolerance float64) bool {
	diff := expected.Sub(actual).Abs()
	return diff.LessThanOrEqual(decimal.NewFromFloat(tolerance))
}
