package fse

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapFSEItems(itemsRequest []structs.FSEItemRequest) ([]fse_models.FSEItem, error) {
	items := make([]fse_models.FSEItem, len(itemsRequest))

	for index, itemReq := range itemsRequest {
		item, err := MapFSEItem(itemReq, index+1)
		if err != nil {
			return nil, err
		}
		items[index] = *item
	}

	return items, nil
}

func MapFSEItem(itemReq structs.FSEItemRequest, index int) (*fse_models.FSEItem, error) {
	baseItem, err := common.MapCommonRequestItem(structs.ItemRequest{
		Number:      index,
		Type:        itemReq.Type,
		Quantity:    itemReq.Quantity,
		Description: itemReq.Description,
		UnitPrice:   itemReq.UnitPrice,
		UnitMeasure: itemReq.UnitMeasure,
		Discount:    itemReq.Discount,
		Code:        itemReq.Code,
	}, index)
	if err != nil {
		return nil, err
	}

	purchase, err := financial.NewAmount(itemReq.Purchase)
	if err != nil {
		return nil, err
	}

	if itemReq.Purchase <= 0 {
		return nil, dte_errors.NewValidationError("RequiredField", "Request->Items->Purchase")
	}

	return fse_models.NewFSEItem(baseItem, *purchase), nil
}
