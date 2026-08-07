package invoice

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapInvoiceResponseItem(items []invoice_models.InvoiceItem) []structs.InvoiceItem {
	result := make([]structs.InvoiceItem, len(items))
	for i, item := range items {
		result[i] = MapInvoiceItem(item)
		result[i].VentaNoSuj = item.NonSubjectSale.GetValue()
		result[i].VentaExenta = item.ExemptSale.GetValue()
		result[i].VentaGravada = item.TaxedSale.GetValue()
		result[i].PSV = item.SuggestedPrice.GetValue()
		result[i].NoGravado = item.NonTaxed.GetValue()
		result[i].IvaItem = item.IVAItem.GetValue()
	}
	return result
}

func MapInvoiceItem(item interfaces.Item) structs.InvoiceItem {
	result := structs.InvoiceItem{
		NumItem:         item.GetNumber(),
		TipoItem:        item.GetType(),
		Descripcion:     item.GetDescription(),
		Cantidad:        item.GetQuantity(),
		UniMedida:       item.GetUnitMeasure(),
		PrecioUni:       item.GetUnitPrice(),
		MontoDescu:      item.GetDiscount(),
		NumeroDocumento: item.GetRelatedDoc(),
	}

	if item.GetTaxes() != nil {
		common.MapTaxCodes(item.GetTaxes())
		result.Tributos = item.GetTaxes()
	} else {
		result.Tributos = nil
	}

	return result
}
