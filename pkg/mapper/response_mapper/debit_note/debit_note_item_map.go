package debit_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

func MapDebitNoteResponseItem(items []debit_note_models.DebitNoteItem) []structs.DebitNoteDTEItem {
	result := make([]structs.DebitNoteDTEItem, len(items))
	for i, item := range items {
		result[i] = structs.DebitNoteDTEItem{
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
		}
	}
	return result
}
