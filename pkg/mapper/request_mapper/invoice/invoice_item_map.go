package invoice

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapInvoiceItems(item []structs.InvoiceItemRequest) ([]invoice_models.InvoiceItem, error) {
	result := make([]invoice_models.InvoiceItem, len(item))

	for i, invoiceItem := range item {
		itemMapped, err := MapInvoiceRequestItem(invoiceItem, i)
		if err != nil {
			return nil, err
		}
		result[i] = *itemMapped
	}

	return result, nil
}

// MapInvoiceRequestItem maps an invoice item -> Source: Request
func MapInvoiceRequestItem(item structs.InvoiceItemRequest, index int) (*invoice_models.InvoiceItem, error) {

	baseItem, err := common.MapCommonRequestItem(structs.ItemRequest{
		Type:        item.Type,
		Quantity:    item.Quantity,
		UnitMeasure: item.UnitMeasure,
		UnitPrice:   item.UnitPrice,
		Discount:    item.Discount,
		Code:        item.Code,
		Taxes:       item.Taxes,
		TaxCode:     item.TaxCode,
		Description: item.Description,
		RelatedDoc:  item.RelatedDoc,
	}, index)

	if err != nil {
		return nil, err
	}

	if item.TaxedSale > 0 && item.IVAItem == 0 {
		return nil, dte_errors.NewValidationError("RequiredField", "Request->Item->IVAItem")
	}

	nonSubjectSale, err := financial.NewAmount(item.NonSubjectSale)
	if err != nil {
		return nil, err
	}

	exemptSale, err := financial.NewAmount(item.ExemptSale)
	if err != nil {
		return nil, err
	}

	taxedSale, err := financial.NewAmount(item.TaxedSale)
	if err != nil {
		return nil, err
	}

	suggestedPrice, err := financial.NewAmount(item.SuggestedPrice)
	if err != nil {
		return nil, err
	}

	nonTaxed, err := financial.NewAmount(item.NonTaxed)
	if err != nil {
		return nil, err
	}

	ivaItem, err := financial.NewAmount(item.IVAItem)
	if err != nil {
		return nil, err
	}

	return &invoice_models.InvoiceItem{
		Item:           baseItem,
		NonSubjectSale: *nonSubjectSale,
		ExemptSale:     *exemptSale,
		TaxedSale:      *taxedSale,
		SuggestedPrice: *suggestedPrice,
		NonTaxed:       *nonTaxed,
		IVAItem:        *ivaItem,
	}, nil
}
