package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
)

// Item is a struct that represents an item of a DTE, containing Number, Type, Code, Description, Quantity,
// UnitMeasure, UnitPrice, Discount, Taxes, TaxCode and RelatedDoc
type Item struct {
	Number      item.ItemNumber    `json:"number"`
	Type        item.ItemType      `json:"type"`
	Description string             `json:"description"`
	Quantity    item.Quantity      `json:"quantity"`
	UnitMeasure item.UnitMeasure   `json:"unitMeasure"`
	UnitPrice   financial.Amount   `json:"unitPrice"`
	Discount    financial.Discount `json:"discount"`
	Taxes       []string           `json:"taxes"`
	Code        *item.ItemCode     `json:"code,omitempty"`
	TaxCode     *financial.TaxType `json:"taxCode,omitempty"`
	RelatedDoc  *string            `json:"relatedDoc,omitempty"`
}

func (i *Item) GetUnitMeasure() int {
	return i.UnitMeasure.GetValue()
}
func (i *Item) GetNumber() int {
	return i.Number.GetValue()
}
func (i *Item) GetQuantity() float64 {
	return i.Quantity.GetValue()
}
func (i *Item) GetItemCode() string {
	return i.Code.GetValue()
}
func (i *Item) GetDescription() string {
	return i.Description
}
func (i *Item) GetType() int {
	return i.Type.GetValue()
}
func (i *Item) GetUnitPrice() float64 {
	return i.UnitPrice.GetValue()
}
func (i *Item) GetDiscount() float64 {
	return i.Discount.GetValue()
}

func (i *Item) GetTaxes() []string {
	if len(i.Taxes) == 0 {
		return nil
	}
	return i.Taxes
}
func (i *Item) GetRelatedDoc() *string {
	return i.RelatedDoc
}

func (i *Item) SetQuantity(quantity float64) error {
	quantityObj, err := item.NewQuantity(quantity)
	if err != nil {
		return err
	}
	i.Quantity = *quantityObj
	return nil
}

func (i *Item) SetItemCode(itemCode string) error {
	if itemCode == "" {
		i.Code = nil
		return nil
	}
	codeObj, err := item.NewItemCode(itemCode)
	if err != nil {
		return err
	}
	i.Code = codeObj
	return nil
}

func (i *Item) SetDescription(description string) error {
	if description == "" {
		return dte_errors.NewValidationError("RequiredField", "Description")
	}
	i.Description = description
	return nil
}

func (i *Item) SetType(itemType int) error {
	typeObj, err := item.NewItemType(itemType)
	if err != nil {
		return err
	}
	i.Type = *typeObj
	return nil
}

func (i *Item) SetUnitPrice(unitPrice float64) error {
	upObj, err := financial.NewAmount(unitPrice)
	if err != nil {
		return err
	}
	i.UnitPrice = *upObj
	return nil
}

func (i *Item) SetForceUnitPrice(unitPrice float64) {
	upObj := financial.NewValidatedAmount(unitPrice)
	i.UnitPrice = *upObj
}

func (i *Item) SetDiscount(discount float64) error {
	discountObj, err := financial.NewDiscount(discount)
	if err != nil {
		return err
	}
	i.Discount = *discountObj
	return nil
}

func (i *Item) SetTaxes(taxes []string) error {
	if len(taxes) == 0 {
		return dte_errors.NewValidationError("RequiredField", "Taxes")
	}

	i.Taxes = taxes
	return nil
}

func (i *Item) SetRelatedDoc(relatedDoc *string) error {
	if relatedDoc == nil {
		return dte_errors.NewValidationError("RequiredField", "RelatedDoc")
	}

	i.RelatedDoc = relatedDoc
	return nil
}

func (i *Item) SetForceRelatedDoc(relatedDoc *string) {
	i.RelatedDoc = relatedDoc
}

func (i *Item) SetNumber(number int) error {
	numberObj, err := item.NewItemNumber(number)
	if err != nil {
		return err
	}
	i.Number = *numberObj
	return nil
}

func (i *Item) SetUnitMeasure(unitMeasure int) error {
	umObj, err := item.NewUnitMeasure(unitMeasure)
	if err != nil {
		return err
	}
	i.UnitMeasure = *umObj
	return nil
}
