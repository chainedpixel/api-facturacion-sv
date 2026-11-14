package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
)

func netResumen(t *testing.T, resumen string) utils.AuxiliarTotalAmountsExtractor {
	t.Helper()
	result, err := utils.ExtractSummaryTotalAmountsFromStringJSON(fmt.Sprintf(`{"resumen":%s}`, resumen))
	require.NoError(t, err)
	return result
}

// TestNetAmountsAdversarial attacks the net-of-discount helpers with boundary and rounding values.
func TestNetAmountsAdversarial(t *testing.T) {
	cases := []struct {
		name    string
		resumen string
		taxed   float64
		exempt  float64
		notSubj float64
	}{
		{"all zero", `{}`, 0, 0, 0},
		{"discount greater than total clamps", `{"totalGravada":5,"descuGravada":9999}`, 0, 0, 0},
		{"discount equal to total", `{"totalExenta":12.34,"descuExenta":12.34}`, 0, 0, 0},
		{"discount without total clamps to zero", `{"descuNoSuj":3}`, 0, 0, 0},
		{"half cent total rounds to cent", `{"totalGravada":0.005}`, 0.01, 0, 0},
		{"half cent discount rounds half away from zero", `{"totalGravada":1,"descuGravada":0.005}`, 1, 0, 0},
		{"three decimals total", `{"totalGravada":33.335,"descuGravada":0}`, 33.34, 0, 0},
		{"large amount no discount", `{"totalGravada":99999999.99}`, 99999999.99, 0, 0},
		{"large amount with discount", `{"totalGravada":99999999.99,"descuGravada":99999999.98}`, 0.01, 0, 0},
		{"float noise 0.1 + 0.2 style", `{"totalGravada":0.3,"descuGravada":0.1}`, 0.2, 0, 0},
		{"float noise 1.15 minus 0.15", `{"totalExenta":1.15,"descuExenta":0.15}`, 0, 1, 0},
		{"null values", `{"totalGravada":null,"descuGravada":null}`, 0, 0, 0},
		{"independent categories", `{"totalGravada":10,"totalExenta":20,"totalNoSuj":30,"descuGravada":1,"descuExenta":2,"descuNoSuj":3}`, 9, 18, 27},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := netResumen(t, tc.resumen)
			assert.InDelta(t, tc.taxed, result.NetTaxed(), 1e-9)
			assert.InDelta(t, tc.exempt, result.NetExempt(), 1e-9)
			assert.InDelta(t, tc.notSubj, result.NetNotSubject(), 1e-9)
		})
	}
}

// TestNetAmountsNeverNegativeOrNaN checks the invariants over a sweep of totals and discounts.
func TestNetAmountsNeverNegativeOrNaN(t *testing.T) {
	values := []float64{0, 0.005, 0.01, 0.99, 1, 33.335, 100, 99999999.99}

	for _, total := range values {
		for _, discount := range values {
			raw := fmt.Sprintf(`{"totalGravada":%v,"descuGravada":%v}`, total, discount)
			result := netResumen(t, raw)
			net := result.NetTaxed()
			assert.GreaterOrEqual(t, net, 0.0, raw)
			assert.LessOrEqual(t, net, total+0.005, raw)
			assert.Equal(t, net, float64(int64(net*100+0.5))/100, "must be rounded to cents: "+raw)
		}
	}
}

// TestExtractSummaryTotalAmountsMalformed checks the extraction errors on invalid summary values.
func TestExtractSummaryTotalAmountsMalformed(t *testing.T) {
	for name, raw := range map[string]string{
		"string discount": `{"resumen":{"descuGravada":"x"}}`,
		"truncated":       `{"resumen":`,
		"empty":           ``,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := utils.ExtractSummaryTotalAmountsFromStringJSON(raw)
			assert.Error(t, err)
		})
	}
}
