package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	itemVO "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// MapCommonRequestItems maps an array of common items to an item model -> Source: Request
func MapCommonRequestItems(items []structs.ItemRequest) ([]models.Item, error) {
	result := make([]models.Item, len(items))

	for i, item := range items {
		itemResult, err := MapCommonRequestItem(item, i)
		if err != nil {
			return nil, err
		}
		result[i] = *itemResult
	}
	return result, nil
}

// MapCommonRequestItem maps a common item to an item model -> Source: Request
func MapCommonRequestItem(item structs.ItemRequest, index int) (*models.Item, error) {
	var err error

	code := itemVO.NewValidatedItemCode("")
	if item.Code != nil {
		code, err = itemVO.NewItemCode(*item.Code)
		if err != nil {
			return nil, err
		}
	}

	taxCode := financial.NewValidatedTaxType("")
	if item.TaxCode != nil {
		taxCode, err = financial.NewTaxType(*item.TaxCode)
		if err != nil {
			return nil, err
		}
	}

	discount, err := financial.NewDiscount(item.Discount)
	if err != nil {
		return nil, err
	}

	quantity, err := itemVO.NewQuantity(item.Quantity)
	if err != nil {
		return nil, err
	}

	unitMeasure, err := itemVO.NewUnitMeasure(item.UnitMeasure)
	if err != nil {
		return nil, err
	}

	unitPrice, err := financial.NewAmount(item.UnitPrice)
	if err != nil {
		return nil, err
	}

	itemType, err := itemVO.NewItemType(item.Type)
	if err != nil {
		return nil, err
	}

	taxes := make([]string, len(item.Taxes))
	if item.Taxes != nil {
		for i, tax := range item.Taxes {
			taxVO, err := financial.NewTaxType(tax)
			if err != nil {
				return nil, err
			}
			taxes[i] = taxVO.GetValue()
		}
	}

	return &models.Item{
		Number:      *itemVO.NewValidatedItemNumber(index + 1),
		Type:        *itemType,
		Quantity:    *quantity,
		UnitMeasure: *unitMeasure,
		UnitPrice:   *unitPrice,
		Discount:    *discount,
		Code:        code,
		Taxes:       taxes,
		TaxCode:     taxCode,
		Description: item.Description,
		RelatedDoc:  item.RelatedDoc,
	}, nil
}
