package services

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/ccf/ccf_models"
	ccfstrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/ccf/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/credit_note/credit_note_models"
	cnstrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/credit_note/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/debit_note/debit_note_models"
	dnstrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/debit_note/validator/strategy"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

// tourismLine is a taxed service line of a document without IVA in the price.
type tourismLine struct {
	sale        float64
	withTourism bool
}

// tourismBaseFixture holds the pieces shared by the CCF, NC and ND models of the tourism base scenarios.
type tourismBaseFixture struct {
	items      []*models.Item
	taxedSales []financial.Amount
	summary    *models.Summary
}

// newTourismBaseFixture builds items and summary: IVA is 13% of the whole taxed total and tax 59 is the declared value.
func newTourismBaseFixture(lines []tourismLine, tourismTax float64) tourismBaseFixture {
	fixture := tourismBaseFixture{}
	taxed := 0.0
	for i, line := range lines {
		codes := []string{constants.TaxIVA}
		if line.withTourism {
			codes = append(codes, constants.TaxTourism)
		}
		fixture.items = append(fixture.items, &models.Item{
			Number: *item.NewValidatedItemNumber(i + 1), Type: *item.NewValidatedItemType(constants.Servicio), Taxes: codes,
		})
		fixture.taxedSales = append(fixture.taxedSales, *financial.NewValidatedAmount(line.sale))
		taxed += line.sale
	}

	iva, _ := decimal.NewFromFloat(taxed).Mul(decimal.NewFromFloat(constants.TaxIvaAmount)).Round(2).Float64()
	newTax := func(code string, value float64) interfaces.Tax {
		return &models.Tax{
			Code: *financial.NewValidatedTaxType(code), Description: "tax",
			Value: &models.TaxAmount{TotalAmount: *financial.NewValidatedAmount(value)},
		}
	}
	total := taxed + iva + tourismTax
	fixture.summary = &models.Summary{
		TotalTaxed:      *financial.NewValidatedAmount(taxed),
		SubTotal:        *financial.NewValidatedAmount(taxed),
		SubTotalSales:   *financial.NewValidatedAmount(taxed),
		TotalTaxes:      []interfaces.Tax{newTax(constants.TaxIVA, iva), newTax(constants.TaxTourism, tourismTax)},
		TotalOperation:  *financial.NewValidatedAmount(total),
		TotalToPay:      *financial.NewValidatedAmount(total),
		TotalNonSubject: *financial.NewValidatedAmount(0),
		TotalExempt:     *financial.NewValidatedAmount(0),
	}
	return fixture
}

// zeroSales returns the zero amounts of the non taxed sale types.
func zeroSales() (financial.Amount, financial.Amount) {
	return *financial.NewValidatedAmount(0), *financial.NewValidatedAmount(0)
}

func (f tourismBaseFixture) ccfValidate() *errorCodeResult {
	items := make([]ccf_models.CreditItem, len(f.items))
	for i := range f.items {
		items[i] = ccf_models.CreditItem{Item: f.items[i], TaxedSale: f.taxedSales[i]}
		items[i].ExemptSale, items[i].NonSubjectSale = zeroSales()
	}
	doc := &ccf_models.CreditFiscalDocument{CreditItems: items, CreditSummary: ccf_models.CreditSummary{Summary: f.summary}}
	return newErrorCodeResult((&ccfstrategy.CCFTaxStrategy{Document: doc}).Validate())
}

func (f tourismBaseFixture) creditNoteValidate() *errorCodeResult {
	items := make([]credit_note_models.CreditNoteItem, len(f.items))
	for i := range f.items {
		items[i] = credit_note_models.CreditNoteItem{Item: f.items[i], TaxedSale: f.taxedSales[i]}
		items[i].ExemptSale, items[i].NonSubjectSale = zeroSales()
	}
	doc := &credit_note_models.CreditNoteModel{CreditItems: items,
		CreditSummary: credit_note_models.CreditNoteSummary{Summary: f.summary}}
	return newErrorCodeResult((&cnstrategy.CreditNoteTaxStrategy{Document: doc}).Validate())
}

func (f tourismBaseFixture) debitNoteValidate() *errorCodeResult {
	items := make([]debit_note_models.DebitNoteItem, len(f.items))
	for i := range f.items {
		items[i] = debit_note_models.DebitNoteItem{Item: f.items[i], TaxedSale: f.taxedSales[i]}
		items[i].ExemptSale, items[i].NonSubjectSale = zeroSales()
	}
	doc := &debit_note_models.DebitNoteModel{DebitItems: items,
		DebitSummary: debit_note_models.DebitNoteSummary{Summary: f.summary}}
	return newErrorCodeResult((&dnstrategy.DebitNoteTaxStrategy{Document: doc}).Validate())
}

// errorCodeResult wraps the outcome of a tax strategy: an empty code means no error.
type errorCodeResult struct{ code string }

func newErrorCodeResult(err *dte_errors.DTEError) *errorCodeResult {
	if err == nil {
		return &errorCodeResult{}
	}
	return &errorCodeResult{code: err.GetCode()}
}

// TestTourismTaxBaseIsTaxedSaleOfItemsCarryingIt checks that CCF, NC and ND compute tax 59 only over the taxed sale of
// the items that carry it, without IVA division, as Hacienda does.
func TestTourismTaxBaseIsTaxedSaleOfItemsCarryingIt(t *testing.T) {
	test.TestMain(t)

	mixed := []tourismLine{{100, true}, {50, false}}
	mixedSmall := []tourismLine{{40, true}, {20, false}}
	all := []tourismLine{{100, true}, {50, true}}

	cases := []struct {
		name    string
		lines   []tourismLine
		tax     float64
		wantErr string
	}{
		{"CCF mixed: 5% of the carrying item", mixed, 5.00, ""},
		{"CCF mixed: 5% of the whole total is rejected", mixed, 7.50, "INVALIDTAXCALCULATION"},
		{"CCF all items carry it: 5% of the total", all, 7.50, ""},
		{"CCF all items carry it: wrong value rejected", all, 5.00, "INVALIDTAXCALCULATION"},
		{"NC/ND mixed: 5% of the carrying item", mixedSmall, 2.00, ""},
		{"NC/ND mixed: 5% of the whole total is rejected", mixedSmall, 3.00, "INVALIDTAXCALCULATION"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newTourismBaseFixture(tc.lines, tc.tax)
			validators := map[string]func() *errorCodeResult{
				"CCF": fixture.ccfValidate, "NC": fixture.creditNoteValidate, "ND": fixture.debitNoteValidate,
			}
			for name, validate := range validators {
				assert.Equal(t, tc.wantErr, validate().code, fmt.Sprintf("%s with 59=%.2f", name, tc.tax))
			}
		})
	}
}

// TestCCFItemStrategyTaxCodeMessageArgumentOrder checks that the message names the tax code first and the item type
// second for a type 4 item with a tax other than 20.
func TestCCFItemStrategyTaxCodeMessageArgumentOrder(t *testing.T) {
	test.TestMain(t)

	doc := &ccf_models.CreditFiscalDocument{CreditItems: []ccf_models.CreditItem{{
		Item: &models.Item{Type: *item.NewValidatedItemType(constants.Impuesto), Taxes: []string{constants.TaxTourism}},
	}}}
	err := (&ccfstrategy.CCFItemStrategy{Document: doc}).Validate()

	require.NotNil(t, err)
	assert.Equal(t, "INVALIDTAXCODEONLY20", err.GetCode())
	assert.Regexp(t, "code 59 .* type 4|c.digo de impuesto 59 .* tipo 4", err.Message)
	assert.NotContains(t, err.Message, "%!")
}
