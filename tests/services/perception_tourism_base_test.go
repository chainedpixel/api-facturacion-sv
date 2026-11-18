package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
)

func perceptionCCFRequest(taxedSale, taxedDiscount, perception float64) *structs.CreateCreditFiscalRequest {
	subTotal := taxedSale - taxedDiscount
	iva := round2(subTotal * 0.13)
	totalOperation := round2(subTotal + iva)
	totalToPay := round2(totalOperation + perception)

	discountPercentage := 0.0
	if taxedDiscount > 0 {
		discountPercentage = round2(taxedDiscount / taxedSale * 100)
	}

	return &structs.CreateCreditFiscalRequest{
		ModelType: 1,
		Receiver:  fixtures.CreateDefaultReceiverWithoutDocsFields(),
		Items: []structs.CreditItemRequest{{
			ItemRequest: structs.ItemRequest{
				Number: 1, Type: 1, Description: "Producto", Quantity: 1,
				UnitMeasure: 59, UnitPrice: taxedSale, Code: itemCode("PER-001"),
				Taxes: []string{"20"},
			},
			TaxedSale: taxedSale,
		}},
		Summary: &structs.CreditSummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				TotalTaxed:         taxedSale,
				SubTotalSales:      taxedSale,
				DiscountPercentage: discountPercentage,
				TotalDiscount:      taxedDiscount,
				SubTotal:           subTotal,
				TotalOperation:     totalOperation,
				TotalToPay:         totalToPay,
				OperationCondition: 1,
				Taxes:              []structs.TaxRequest{{Code: "20", Description: "Impuesto al Valor Agregado 13%", Value: iva}},
				PaymentTypes:       []structs.PaymentRequest{{Code: "01", Amount: totalToPay}},
			},
			TaxedDiscount: taxedDiscount,
			IVAPerception: perception,
		},
	}
}

func round2(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}

func TestIVAPerceptionBaseIsTaxedTotalAfterGlobalDiscount(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name          string
		taxedSale     float64
		taxedDiscount float64
		perception    float64
		wantErr       string
	}{
		{"without discount, 1% of taxed total", 700, 0, 7.00, ""},
		{"without discount, wrong amount", 700, 0, 6.00, "InvalidPerceptionAmount"},
		{"with discount, 1% of discounted base (real document)", 700, 50, 6.50, ""},
		{"with discount, pre-discount base is rejected (real document)", 700, 50, 7.00, "InvalidPerceptionAmount"},
		{"with discount, rounding to cents", 333.33, 33.33, 3.00, ""},
		{"with discount, rounding up", 105.55, 5, 1.01, ""},
		{"zero perception is not validated", 700, 50, 0, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := createCCF(t, perceptionCCFRequest(tc.taxedSale, tc.taxedDiscount, tc.perception))
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func tourismInvoiceRequest(ventaGravada, tourismTax float64) *structs.CreateInvoiceRequest {
	iva := round2(ventaGravada / 1.13 * 0.13)
	totalOperation := round2(ventaGravada + tourismTax)

	return &structs.CreateInvoiceRequest{
		ModelType: 1,
		Receiver:  fixtures.CreateDefaultReceiver(),
		Items: []structs.InvoiceItemRequest{{
			ItemRequest: structs.ItemRequest{
				Number: 1, Type: 1, Description: "Alojamiento", Quantity: 1,
				UnitMeasure: 59, UnitPrice: ventaGravada, Code: itemCode("HOT-001"),
				Taxes: []string{"59"},
			},
			TaxedSale: ventaGravada,
			IVAItem:   iva,
		}},
		Summary: &structs.InvoiceSummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				TotalTaxed:         ventaGravada,
				SubTotalSales:      ventaGravada,
				SubTotal:           ventaGravada,
				TotalOperation:     totalOperation,
				TotalToPay:         totalOperation,
				OperationCondition: 1,
				Taxes:              []structs.TaxRequest{{Code: "59", Description: "Turismo: por alojamiento (5%)", Value: tourismTax}},
				PaymentTypes:       []structs.PaymentRequest{{Code: "01", Amount: totalOperation}},
			},
			TotalIVA: iva,
		},
	}
}

func TestTourismTaxInInvoiceExcludesIVAFromBase(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name         string
		ventaGravada float64
		tourismTax   float64
		wantErr      string
	}{
		{"real document, base without IVA", 100, 4.42, ""},
		{"real document, base with IVA is rejected", 100, 5.00, "InvalidTaxCalculation"},
		{"rounding down 75.50", 75.50, 3.34, ""},
		{"rounding 75.50, IVA-inclusive base is rejected", 75.50, 3.78, "InvalidTaxCalculation"},
		{"rounding up 113", 113, 5.00, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := createInvoice(t, tourismInvoiceRequest(tc.ventaGravada, tc.tourismTax))
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, strings.ToUpper(err.Error()), strings.ToUpper(tc.wantErr))
		})
	}
}

func TestTourismTaxInCCFKeepsIVAExclusiveBase(t *testing.T) {
	test.TestMain(t)

	build := func(tourismTax float64) *structs.CreateCreditFiscalRequest {
		req := perceptionCCFRequest(100, 0, 0)
		req.Items[0].Type = 2
		req.Items[0].Taxes = []string{"20", "59"}
		req.Summary.Taxes = append(req.Summary.Taxes,
			structs.TaxRequest{Code: "59", Description: "Turismo: por alojamiento (5%)", Value: tourismTax})
		total := round2(100 + 13 + tourismTax)
		req.Summary.TotalOperation = total
		req.Summary.TotalToPay = total
		req.Summary.PaymentTypes[0].Amount = total
		return req
	}

	for _, tc := range []struct {
		tax     float64
		wantErr bool
	}{{5.00, false}, {4.42, true}} {
		t.Run(fmt.Sprintf("tourism %.2f", tc.tax), func(t *testing.T) {
			err := createCCF(t, build(tc.tax))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// tourismItem is a taxed invoice item for the tourism tax scenarios
type tourismItem struct {
	sale        float64
	withTourism bool
}

// tourismScenarioInvoice builds an FE where the tourism tax base is the taxed sale without IVA of the items that carry
// it, ignoring the global discount on taxed sales
func tourismScenarioInvoice(items []tourismItem, taxedDiscount, tourismTax, totalIVA float64) *structs.CreateInvoiceRequest {
	var taxed float64
	requestItems := make([]structs.InvoiceItemRequest, 0, len(items))
	for i, it := range items {
		taxes := []string(nil)
		if it.withTourism {
			taxes = []string{"59"}
		}
		taxed += it.sale
		requestItems = append(requestItems, structs.InvoiceItemRequest{
			ItemRequest: structs.ItemRequest{
				Number: i + 1, Type: 1, Description: "Item", Quantity: 1, UnitMeasure: 59, UnitPrice: it.sale,
				Code: itemCode(fmt.Sprintf("TUR-%03d", i+1)), Taxes: taxes,
			},
			TaxedSale: it.sale,
			IVAItem:   round2(it.sale / 1.13 * 0.13),
		})
	}

	subTotal := taxed - taxedDiscount
	totalOperation := round2(subTotal + tourismTax)
	percentage := 0.0
	if taxedDiscount > 0 {
		percentage = round2(taxedDiscount / taxed * 100)
	}

	return &structs.CreateInvoiceRequest{
		ModelType: 1,
		Receiver:  fixtures.CreateDefaultReceiver(),
		Items:     requestItems,
		Summary: &structs.InvoiceSummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				TotalTaxed:         taxed,
				SubTotalSales:      taxed,
				DiscountPercentage: percentage,
				TotalDiscount:      taxedDiscount,
				SubTotal:           subTotal,
				TotalOperation:     totalOperation,
				TotalToPay:         totalOperation,
				OperationCondition: 1,
				Taxes:              []structs.TaxRequest{{Code: "59", Description: "Turismo: por alojamiento (5%)", Value: tourismTax}},
				PaymentTypes:       []structs.PaymentRequest{{Code: "01", Amount: totalOperation}},
			},
			TaxedDiscount: taxedDiscount,
			TotalIVA:      totalIVA,
		},
	}
}

func TestTourismTaxInInvoiceMixedItemsAndGlobalDiscount(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name          string
		items         []tourismItem
		taxedDiscount float64
		tourismTax    float64
		totalIVA      float64
		wantErr       string
	}{
		{"mixed: only the item with the tax is the base", []tourismItem{{100, true}, {50, false}}, 0, 4.42, 17.26, ""},
		{"mixed: base over every item is rejected", []tourismItem{{100, true}, {50, false}}, 0, 6.64, 17.26, "InvalidTaxCalculation"},
		{"global discount is ignored by the base", []tourismItem{{100, true}}, 10, 4.42, 10.35, ""},
		{"global discount applied to the base is rejected", []tourismItem{{100, true}}, 10, 3.98, 10.35, "InvalidTaxCalculation"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := createInvoice(t, tourismScenarioInvoice(tc.items, tc.taxedDiscount, tc.tourismTax, tc.totalIVA))
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, strings.ToUpper(err.Error()), strings.ToUpper(tc.wantErr))
		})
	}
}

func TestTourismTaxSummaryConsistencyInInvoice(t *testing.T) {
	test.TestMain(t)

	withoutSummaryTax := func(req *structs.CreateInvoiceRequest) *structs.CreateInvoiceRequest {
		req.Summary.Taxes = nil
		return req
	}

	cases := []struct {
		name    string
		request *structs.CreateInvoiceRequest
		wantErr string
	}{
		{"item with 59 and no summary entry", withoutSummaryTax(tourismScenarioInvoice([]tourismItem{{100, true}}, 0, 0, 11.50)), "MissingSummaryTax"},
		{"mixed items, only the carrying item and no summary entry", withoutSummaryTax(tourismScenarioInvoice([]tourismItem{{100, true}, {50, false}}, 0, 0, 17.26)), "MissingSummaryTax"},
		{"summary entry 59 and no item carrying it", tourismScenarioInvoice([]tourismItem{{100, false}}, 0, 4.42, 11.50), "MissingItemTaxForSummary"},
		{"both present and correct", tourismScenarioInvoice([]tourismItem{{100, true}}, 0, 4.42, 11.50), ""},
		{"mixed items, summary computed over the carrying item only", tourismScenarioInvoice([]tourismItem{{100, true}, {50, false}}, 0, 4.42, 17.26), ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := createInvoice(t, tc.request)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, strings.ToUpper(err.Error()), strings.ToUpper(tc.wantErr))
		})
	}
}

func TestTourismTaxSummaryConsistencyInCCF(t *testing.T) {
	test.TestMain(t)

	build := func(itemTaxes []string, withSummaryTax bool) *structs.CreateCreditFiscalRequest {
		req := perceptionCCFRequest(100, 0, 0)
		req.Items[0].Type = 2
		req.Items[0].Taxes = itemTaxes
		total := 113.0
		if withSummaryTax {
			req.Summary.Taxes = append(req.Summary.Taxes,
				structs.TaxRequest{Code: "59", Description: "Turismo: por alojamiento (5%)", Value: 5})
			total = 118
		}
		req.Summary.TotalOperation = total
		req.Summary.TotalToPay = total
		req.Summary.PaymentTypes[0].Amount = total
		return req
	}

	cases := []struct {
		name    string
		request *structs.CreateCreditFiscalRequest
		wantErr string
	}{
		{"item with 59 and no summary entry", build([]string{"20", "59"}, false), "MissingSummaryTax"},
		{"summary entry 59 and no item carrying it", build([]string{"20"}, true), "MissingItemTaxForSummary"},
		{"both present and correct", build([]string{"20", "59"}, true), ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := createCCF(t, tc.request)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, strings.ToUpper(err.Error()), strings.ToUpper(tc.wantErr))
		})
	}
}
