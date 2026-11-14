package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/validator/strategy"
	test "github.com/chainedpixel/ordo-factus/tests"
)

type adversarialTax struct {
	code  string
	value float64
}

func adversarialDebitNote(taxed float64, taxes []adversarialTax) *debit_note_models.DebitNoteModel {
	summaryTaxes := make([]interfaces.Tax, 0, len(taxes))
	taxTotal := 0.0
	for _, tx := range taxes {
		taxEntity := &models.Tax{
			Code:        *financial.NewValidatedTaxType(tx.code),
			Description: "tax",
			Value:       &models.TaxAmount{TotalAmount: *financial.NewValidatedAmount(tx.value)},
		}
		summaryTaxes = append(summaryTaxes, taxEntity)
		taxTotal += tx.value
	}

	item := debit_note_models.DebitNoteItem{
		Item:           &models.Item{},
		TaxedSale:      *financial.NewValidatedAmount(taxed),
		ExemptSale:     *financial.NewValidatedAmount(0),
		NonSubjectSale: *financial.NewValidatedAmount(0),
	}

	return &debit_note_models.DebitNoteModel{
		DebitItems: []debit_note_models.DebitNoteItem{item},
		DebitSummary: debit_note_models.DebitNoteSummary{
			Summary: &models.Summary{
				TotalTaxed:      *financial.NewValidatedAmount(taxed),
				SubTotal:        *financial.NewValidatedAmount(taxed),
				SubTotalSales:   *financial.NewValidatedAmount(taxed),
				TotalTaxes:      summaryTaxes,
				TotalOperation:  *financial.NewValidatedAmount(taxed + taxTotal),
				TotalToPay:      *financial.NewValidatedAmount(taxed + taxTotal),
				TotalNonSubject: *financial.NewValidatedAmount(0),
				TotalExempt:     *financial.NewValidatedAmount(0),
			},
		},
	}
}

// TestDebitNoteTaxStrategyAdversarial attacks the tolerance, rounding and tax-code branches of validateIVA.
func TestDebitNoteTaxStrategyAdversarial(t *testing.T) {
	test.TestMain(t)

	iva := constants.TaxIVA
	cases := []struct {
		name      string
		taxed     float64
		taxes     []adversarialTax
		errorCode string
	}{
		{"exact IVA", 100, []adversarialTax{{iva, 13}}, ""},
		{"IVA +0.01 boundary accepted", 100, []adversarialTax{{iva, 13.01}}, ""},
		{"IVA -0.01 boundary accepted", 100, []adversarialTax{{iva, 12.99}}, ""},
		{"IVA +0.011 rejected", 100, []adversarialTax{{iva, 13.011}}, "INVALIDTAXCALCULATION"},
		{"IVA -0.011 rejected", 100, []adversarialTax{{iva, 12.989}}, "INVALIDTAXCALCULATION"},
		{"IVA off by 1.00 rejected", 100, []adversarialTax{{iva, 14}}, "INVALIDTAXCALCULATION"},
		{"IVA zero with taxed rejected", 100, []adversarialTax{{iva, 0}}, "INVALIDTAXCALCULATION"},
		{"tiny taxed 0.05 with IVA 0.01", 0.05, []adversarialTax{{iva, 0.01}}, ""},
		{"fractional 33.335 correct rounded IVA", 33.335, []adversarialTax{{iva, 4.33}}, ""},
		{"fractional 33.335 wrong IVA", 33.335, []adversarialTax{{iva, 4.35}}, "INVALIDTAXCALCULATION"},
		{"large amount exact", 99999999.99, []adversarialTax{{iva, 13000000.00}}, ""},
		{"large amount off by 0.02", 99999999.99, []adversarialTax{{iva, 13000000.02}}, "INVALIDTAXCALCULATION"},
		{"tourism tax correct plus IVA", 100, []adversarialTax{{iva, 13}, {constants.TaxTourism, 5}}, ""},
		{"tourism tax wrong", 100, []adversarialTax{{iva, 13}, {constants.TaxTourism, 6}}, "INVALIDTAXCALCULATION"},
		{"airport tax fixed amount", 100, []adversarialTax{{iva, 13}, {constants.TaxTourismAirport, 7}}, ""},
		{"airport tax wrong", 100, []adversarialTax{{iva, 13}, {constants.TaxTourismAirport, 8}}, "INVALIDTAXCALCULATION"},
		{"export IVA with positive value rejected", 100, []adversarialTax{{iva, 13}, {constants.TaxIVAExport, 1}}, "INVALIDTAXCALCULATION"},
		{"unknown tax code with value rejected", 100, []adversarialTax{{iva, 13}, {"ZZ", 5}}, "INVALIDTAXCALCULATION"},
		{"special other skipped", 100, []adversarialTax{{iva, 13}, {constants.TaxSpecialOther, 5}}, ""},
		{"zero taxed without taxes", 0, nil, ""},
		{"zero taxed with taxes", 0, []adversarialTax{{iva, 0.13}}, "INVALIDTAXES"},
		{"taxed without taxes", 100, nil, "MISSINGTAXES"},
		{"only special other with taxed", 100, []adversarialTax{{constants.TaxSpecialOther, 5}}, "MISSINGIVAFORTAXEDAMOUNT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := adversarialDebitNote(tc.taxed, tc.taxes)
			err := (&strategy.DebitNoteTaxStrategy{Document: doc}).Validate()
			if tc.errorCode == "" {
				assert.Nil(t, err)
				return
			}
			if assert.NotNil(t, err) {
				assert.Equal(t, tc.errorCode, err.GetCode())
				assert.NotContains(t, err.Message, "No se encontr")
				assert.NotContains(t, strings.ToLower(err.Message), "unknownerror")
				assert.NotContains(t, err.Message, "validation_errors.")
			}
		})
	}
}

// TestDebitNoteInvalidTaxCalculationMessageParams checks that the message carries the tax code and both amounts.
func TestDebitNoteInvalidTaxCalculationMessageParams(t *testing.T) {
	test.TestMain(t)

	doc := adversarialDebitNote(100, []adversarialTax{{constants.TaxIVA, 14}})
	err := (&strategy.DebitNoteTaxStrategy{Document: doc}).Validate()
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Message, "20")
		assert.Contains(t, err.Message, "14")
		assert.NotContains(t, err.Message, "%!")
		assert.NotContains(t, err.Message, "MISSING")
	}
}
