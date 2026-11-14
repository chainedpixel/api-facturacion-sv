package mappers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/common"
)

// Se verifica que codigo y codTributo lleguen a la estructura de Hacienda (antes codigo se enviaba null)
func TestMapItemCodes(t *testing.T) {
	tests := []struct {
		name        string
		item        *models.Item
		wantCode    *string
		wantTaxCode *string
	}{
		{
			name:     "item de producto con codigo",
			item:     &models.Item{Type: *item.NewValidatedItemType(constants.Producto), Code: item.NewValidatedItemCode("PROD-001")},
			wantCode: ptr("PROD-001"),
		},
		{
			name: "item sin codigo se envia null",
			item: &models.Item{Type: *item.NewValidatedItemType(constants.Producto), Code: item.NewValidatedItemCode("")},
		},
		{
			name: "item con Code nil no entra en panico",
			item: &models.Item{Type: *item.NewValidatedItemType(constants.Producto)},
		},
		{
			name:     "codTributo se ignora fuera de items tipo 4",
			item:     &models.Item{Type: *item.NewValidatedItemType(constants.Producto), TaxCode: financial.NewValidatedTaxType("A8")},
			wantCode: nil,
		},
		{
			name:        "item tipo 4 lleva codTributo",
			item:        &models.Item{Type: *item.NewValidatedItemType(constants.Impuesto), Code: item.NewValidatedItemCode("IMP-1"), TaxCode: financial.NewValidatedTaxType("A8")},
			wantCode:    ptr("IMP-1"),
			wantTaxCode: ptr("A8"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, taxCode := common.MapItemCodes(tt.item)
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantTaxCode, taxCode)
		})
	}
}

func ptr(value string) *string { return &value }
