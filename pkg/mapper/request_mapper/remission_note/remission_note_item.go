package remission_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapRemissionNoteItems(items []*structs.RemissionNoteItemRequest) ([]remission_note_models.RemissionNoteItem, error) {
	if items == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Items")
	}

	remissionItems := make([]remission_note_models.RemissionNoteItem, len(items))

	for i, item := range items {
		if item == nil {
			return nil, dte_errors.NewValidationError("RequiredField", "Item")
		}

		remissionItem, err := mapSingleRemissionNoteItem(item)
		if err != nil {
			return nil, err
		}

		remissionItems[i] = *remissionItem
	}

	return remissionItems, nil
}

func mapSingleRemissionNoteItem(item *structs.RemissionNoteItemRequest) (*remission_note_models.RemissionNoteItem, error) {
	if err := validateRemissionNoteItem(item); err != nil {
		return nil, err
	}

	baseItem := &models.Item{}

	if err := baseItem.SetNumber(*item.ItemNumber); err != nil {
		return nil, err
	}
	if err := baseItem.SetType(*item.ItemType); err != nil {
		return nil, err
	}
	if err := baseItem.SetDescription(*item.Description); err != nil {
		return nil, err
	}
	if err := baseItem.SetQuantity(*item.Quantity); err != nil {
		return nil, err
	}
	if err := baseItem.SetUnitMeasure(*item.UnitMeasure); err != nil {
		return nil, err
	}
	if err := baseItem.SetUnitPrice(*item.UnitPrice); err != nil {
		return nil, err
	}

	if item.Code != nil {
		if err := baseItem.SetItemCode(*item.Code); err != nil {
			return nil, err
		}
	}

	if item.DiscountAmount != nil {
		if err := baseItem.SetDiscount(*item.DiscountAmount); err != nil {
			return nil, err
		}
	}

	if len(item.Tributes) > 0 {
		if err := baseItem.SetTaxes(item.Tributes); err != nil {
			return nil, err
		}
	}

	nonSubjectSale, err := financial.NewAmount(getValueOrZero(item.NonSubjectSale))
	if err != nil {
		return nil, err
	}

	exemptSale, err := financial.NewAmount(getValueOrZero(item.ExemptSale))
	if err != nil {
		return nil, err
	}

	taxedSale, err := financial.NewAmount(getValueOrZero(item.TaxedSale))
	if err != nil {
		return nil, err
	}

	return &remission_note_models.RemissionNoteItem{
		Item:           baseItem,
		NonSubjectSale: *nonSubjectSale,
		ExemptSale:     *exemptSale,
		TaxedSale:      *taxedSale,
	}, nil
}

func validateRemissionNoteItem(item *structs.RemissionNoteItemRequest) error {
	if item.ItemNumber == nil {
		return dte_errors.NewValidationError("RequiredField", "Item->ItemNumber")
	}

	if item.ItemType == nil {
		return dte_errors.NewValidationError("RequiredField", "Item->ItemType")
	}

	if item.Description == nil || *item.Description == "" {
		return dte_errors.NewValidationError("RequiredField", "Item->Description")
	}

	if item.Quantity == nil {
		return dte_errors.NewValidationError("RequiredField", "Item->Quantity")
	}

	if item.UnitMeasure == nil {
		return dte_errors.NewValidationError("RequiredField", "Item->UnitMeasure")
	}

	if item.UnitPrice == nil {
		return dte_errors.NewValidationError("RequiredField", "Item->UnitPrice")
	}

	nonSubject := getValueOrZero(item.NonSubjectSale)
	exempt := getValueOrZero(item.ExemptSale)
	taxed := getValueOrZero(item.TaxedSale)

	if nonSubject == 0 && exempt == 0 && taxed == 0 {
		return dte_errors.NewValidationError("InvalidValue", "At least one sale type must have value")
	}

	return nil
}

func getValueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
