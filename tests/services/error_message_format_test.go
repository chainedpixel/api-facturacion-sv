package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	ccfStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
)

// TestCCFErrorMessagesFormat checks that CCF calculation errors render their values in the expected places.
func TestCCFErrorMessagesFormat(t *testing.T) {
	test.TestMain(t)

	t.Run("Perception amount is rendered with two decimals", func(t *testing.T) {
		ccf, err := fixtures.BuildCCF()
		assert.NoError(t, err)
		taxed := ccf.CreditSummary.Summary.GetTotalTaxed()
		ccf.CreditSummary.IVAPerception = *financial.NewValidatedAmount(taxed * 0.02)

		dteErr := (&ccfStrategy.CCFTaxStrategy{Document: ccf}).Validate()
		if assert.NotNil(t, dteErr) {
			message := dteErr.Error()
			assert.NotContains(t, message, "%!")
			assert.Contains(t, message, decimal.NewFromFloat(taxed*0.02).StringFixed(2))
			assert.Contains(t, message, decimal.NewFromFloat(taxed).Mul(decimal.RequireFromString("0.01")).StringFixed(2))
		}
	})

	t.Run("Tax calculation shows expected before actual", func(t *testing.T) {
		ccf, err := fixtures.BuildCCF()
		assert.NoError(t, err)

		var expected float64
		for _, item := range ccf.CreditSummary.Summary.GetTotalTaxes() {
			if tax, ok := item.(*models.Tax); ok && tax.GetCode() == constants.TaxIVA {
				expected = tax.GetValue()
				assert.NoError(t, tax.SetValue(expected+50))
			}
		}

		dteErr := (&ccfStrategy.CCFTaxStrategy{Document: ccf}).Validate()
		if assert.NotNil(t, dteErr) {
			message := dteErr.Error()
			var reportedExpected, reportedActual float64
			_, scanErr := fmt.Sscanf(message[strings.Index(message, "Expected:"):], "Expected: %f, Actual: %f", &reportedExpected, &reportedActual)
			assert.NoError(t, scanErr)
			assert.InDelta(t, expected, reportedExpected, 0.01)
			assert.InDelta(t, expected+50, reportedActual, 0.000001)
		}
	})
}
