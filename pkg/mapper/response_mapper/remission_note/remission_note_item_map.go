package remission_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapRemissionNoteItems converts the domain items to the Ministry of Finance format
func MapRemissionNoteItems(model *remission_note_models.RemissionNoteModel) []*structs.MHRemissionNoteItem {
	if len(model.RemissionItems) == 0 {
		return nil
	}

	mhItems := make([]*structs.MHRemissionNoteItem, len(model.RemissionItems))

	for i, item := range model.RemissionItems {
		mhItems[i] = mapSingleRemissionNoteItem(&item)
	}

	return mhItems
}

func mapSingleRemissionNoteItem(item *remission_note_models.RemissionNoteItem) *structs.MHRemissionNoteItem {
	mhItem := &structs.MHRemissionNoteItem{
		ItemNumber:     item.Item.GetNumber(),
		ItemType:       item.Item.GetType(),
		Description:    item.Item.GetDescription(),
		Quantity:       item.Item.GetQuantity(),
		UnitMeasure:    item.Item.GetUnitMeasure(),
		UnitPrice:      item.Item.GetUnitPrice(),
		DiscountAmount: item.Item.GetDiscount(),
		NonSubjectSale: item.NonSubjectSale.GetValue(),
		ExemptSale:     item.ExemptSale.GetValue(),
		TaxedSale:      item.TaxedSale.GetValue(),
		Tributes:       item.Item.GetTaxes(),
	}

	if relatedDoc := item.Item.GetRelatedDoc(); relatedDoc != nil {
		mhItem.DocumentNumber = relatedDoc
	}

	if code := item.Item.GetItemCode(); code != "" {
		mhItem.Code = &code
	}

	if taxes := item.Item.GetTaxes(); len(taxes) > 0 {
		tributeCode := taxes[0]
		mhItem.TributeCode = &tributeCode
	}

	return mhItem
}

func getItemNumber(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func getItemType(value *int) int {
	if value == nil {
		return 1
	}
	return *value
}

func getStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func getFloatValue(value *float64) float64 {
	if value == nil {
		return 0.0
	}
	return *value
}

func getIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func getDiscountAmount(amount interface{}) float64 {
	if amount == nil {
		return 0.0
	}
	if financialAmount, ok := amount.(interface{ GetValue() float64 }); ok {
		return financialAmount.GetValue()
	}
	return 0.0
}
