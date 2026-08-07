package retention

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapRetentionResponseItem(items []retention_models.RetentionItem) []structs.RetentionItem {
	if items == nil {
		return nil
	}

	retentionItems := make([]structs.RetentionItem, len(items))
	for i, item := range items {
		retentionItems[i] = structs.RetentionItem{
			NumItem:            item.Number.GetValue(),
			TipoDTE:            item.DTEType.GetValue(),
			TipoDoc:            item.DocumentType.GetValue(),
			NumDoc:             item.DocumentNumber.GetValue(),
			FechaEmision:       item.EmissionDate.GetValue().Format("2006-01-02"),
			MontoSujetoGravado: item.RetentionAmount.GetValue(),
			CodigoRetencionMH:  item.ReceptionCodeMH.GetValue(),
			IvaRetenido:        item.RetentionIVA.GetValue(),
			Descripcion:        item.Description,
		}
	}

	return retentionItems
}
