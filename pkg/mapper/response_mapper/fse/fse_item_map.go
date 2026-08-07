package fse

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapFSEResponseItems(items []fse_models.FSEItem) []structs.FSEItemResponse {
	if len(items) == 0 {
		return []structs.FSEItemResponse{}
	}

	responseItems := make([]structs.FSEItemResponse, len(items))

	for i, item := range items {
		responseItems[i] = MapFSEResponseItemWithNumber(item, i+1)
	}

	return responseItems
}

func MapFSEResponseItem(item fse_models.FSEItem) structs.FSEItemResponse {
	return MapFSEResponseItemWithNumber(item, item.Number.GetValue())
}

func MapFSEResponseItemWithNumber(item fse_models.FSEItem, itemNumber int) structs.FSEItemResponse {
	var itemCode *string
	if item.Code != nil && item.Code.IsValid() {
		code := item.Code.GetValue()
		itemCode = &code
	}

	return structs.FSEItemResponse{
		NumItem:     itemNumber,
		TipoItem:    item.Type.GetValue(),
		Cantidad:    item.Quantity.GetValue(),
		Codigo:      itemCode,
		UniMedida:   item.UnitMeasure.GetValue(),
		Descripcion: item.Description,
		PrecioUni:   item.UnitPrice.GetValue(),
		MontoDescu:  item.Discount.GetValue(),
		Compra:      item.Purchase.GetValue(),
	}
}
