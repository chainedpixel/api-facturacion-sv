package credit_note

import (
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

func MapCreditNoteResponseItem(items []credit_note_models.CreditNoteItem) []structs.CreditNoteDTEItem {
	result := make([]structs.CreditNoteDTEItem, len(items))
	for i, item := range items {
		var totalIva float64
		if item.TaxedSale.GetValue() > 0 {
			totalIva = math.Round(item.TaxedSale.GetValue()*0.13*100) / 100
		}

		result[i] = structs.CreditNoteDTEItem{
			NumItem:         item.GetNumber(),
			TipoItem:        item.GetType(),
			NumeroDocumento: item.GetRelatedDoc(),
			CodTributo:      utils.ToStringPointer(item.TaxCode.GetValue()),
			Codigo:          utils.ToStringPointer(item.GetItemCode()),
			Descripcion:     item.GetDescription(),
			Cantidad:        item.GetQuantity(),
			UniMedida:       item.GetUnitMeasure(),
			PrecioUni:       item.GetUnitPrice(),
			MontoDescu:      item.GetDiscount(),
			VentaNoSuj:      item.NonSubjectSale.GetValue(),
			VentaExenta:     item.ExemptSale.GetValue(),
			VentaGravada:    item.TaxedSale.GetValue(),
			Tributos:        item.GetTaxes(),
			NoGravado:       0,
			IvaPerci:        0,
			TotalIva:        totalIva,
			IvaRete:         0,
		}
	}
	return result
}
