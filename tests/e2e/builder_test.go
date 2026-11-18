//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tourismWire is the JSON shape of an FE request that the tourism assertions need
type tourismWire struct {
	Items []struct {
		Taxes []string `json:"taxes"`
	} `json:"items"`
	Summary struct {
		Taxes []struct {
			Code  string  `json:"code"`
			Value float64 `json:"value"`
		} `json:"taxes"`
		TotalTaxed     float64 `json:"total_taxed"`
		SubTotal       float64 `json:"sub_total"`
		TotalOperation float64 `json:"total_operation"`
		TotalToPay     float64 `json:"total_to_pay"`
		TotalIVA       float64 `json:"total_iva"`
		PaymentTypes   []struct {
			Amount float64 `json:"amount"`
		} `json:"payment_types"`
	} `json:"summary"`
}

// TestBuilder_TourismTaxRequests checks offline that the tourism FE requests built for the E2E suite are well formed.
func TestBuilder_TourismTaxRequests(t *testing.T) {
	expected := map[string]struct {
		itemTaxes            [][]string
		tax, taxed, subTotal float64
		operation, totalIVA  float64
	}{
		"tributo adicional turismo 59": {
			itemTaxes: [][]string{{"59"}}, tax: 4.42, taxed: 100, subTotal: 100, operation: 104.42, totalIVA: 11.50},
		"turismo 59 mixto: un item con tributo y otro sin tributo": {
			itemTaxes: [][]string{{"59"}, nil}, tax: 4.42, taxed: 150, subTotal: 150, operation: 154.42, totalIVA: 17.26},
		"turismo 59 con descuento global a ventas gravadas": {
			itemTaxes: [][]string{{"59"}}, tax: 4.42, taxed: 100, subTotal: 90, operation: 94.42, totalIVA: 10.35},
	}

	cases := tourismInvoiceCases()
	require.Len(t, cases, len(expected))

	for _, tc := range cases {
		want, ok := expected[tc.name]
		require.True(t, ok, tc.name)

		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(buildInvoice(tc.spec))
			require.NoError(t, err)
			var got tourismWire
			require.NoError(t, json.Unmarshal(raw, &got))

			require.Len(t, got.Items, len(want.itemTaxes))
			for i, taxes := range want.itemTaxes {
				assert.Equal(t, taxes, got.Items[i].Taxes, "item %d taxes", i+1)
			}
			require.Len(t, got.Summary.Taxes, 1)
			assert.Equal(t, "59", got.Summary.Taxes[0].Code)
			assert.Equal(t, want.tax, got.Summary.Taxes[0].Value)
			assert.Equal(t, want.taxed, got.Summary.TotalTaxed)
			assert.Equal(t, want.subTotal, got.Summary.SubTotal)
			assert.Equal(t, want.operation, got.Summary.TotalOperation)
			assert.Equal(t, want.operation, got.Summary.TotalToPay)
			assert.Equal(t, want.totalIVA, got.Summary.TotalIVA)
			require.Len(t, got.Summary.PaymentTypes, 1)
			assert.Equal(t, want.operation, got.Summary.PaymentTypes[0].Amount)
			t.Logf("items=%v tax=%.2f total_taxed=%.2f sub_total=%.2f total_operation=%.2f total_to_pay=%.2f payment=%.2f total_iva=%.2f",
				want.itemTaxes, got.Summary.Taxes[0].Value, got.Summary.TotalTaxed, got.Summary.SubTotal,
				got.Summary.TotalOperation, got.Summary.TotalToPay, got.Summary.PaymentTypes[0].Amount, got.Summary.TotalIVA)
		})
	}
}
