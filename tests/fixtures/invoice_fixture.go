package fixtures

import (
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	respStructs "github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CreateDefaultInvoiceItem creates a valid default invoice item
func CreateDefaultInvoiceItem(index int) structs.InvoiceItemRequest {
	code := "COD" + string(rune(65+index))

	return structs.InvoiceItemRequest{
		ItemRequest: structs.ItemRequest{
			Number:      index + 1,
			Type:        1,
			Description: "Producto " + string(rune(65+index)),
			Quantity:    10,
			UnitMeasure: 59,
			UnitPrice:   5.0,
			Discount:    0,
			Code:        &code,
			Taxes:       []string{"20"},
		},
		NonSubjectSale: 0,
		ExemptSale:     0,
		TaxedSale:      50.0,
		SuggestedPrice: 0,
		NonTaxed:       0,
		IVAItem:        6.5,
	}
}

// CreateInvoiceItemWithInvalidType creates an invoice item with an invalid type
func CreateInvoiceItemWithInvalidType() structs.InvoiceItemRequest {
	item := CreateDefaultInvoiceItem(0)
	item.Type = 99
	return item
}

// CreateInvoiceItemWithNegativeQuantity creates an invoice item with a negative quantity
func CreateInvoiceItemWithNegativeQuantity() structs.InvoiceItemRequest {
	item := CreateDefaultInvoiceItem(0)
	item.Quantity = -1
	return item
}

// CreateInvoiceItemWithInvalidIVA creates an invoice item with an invalid IVA
func CreateInvoiceItemWithInvalidIVA() structs.InvoiceItemRequest {
	item := CreateDefaultInvoiceItem(0)
	item.IVAItem = 100
	return item
}

// CreateDefaultInvoiceSummary creates a valid default invoice summary
func CreateDefaultInvoiceSummary() *structs.InvoiceSummaryRequest {
	return &structs.InvoiceSummaryRequest{
		SummaryRequest: structs.SummaryRequest{
			TotalNonSubject:    0,
			TotalExempt:        0,
			TotalTaxed:         100.0,
			SubTotal:           100.0,
			NonSubjectDiscount: 0,
			ExemptDiscount:     0,
			DiscountPercentage: 0,
			TotalDiscount:      0,
			SubTotalSales:      100.0,
			TotalOperation:     100.0,
			TotalNonTaxed:      0,
			TotalToPay:         100.0,
			OperationCondition: 1,
			Taxes: []structs.TaxRequest{
				{
					Code:        "20",
					Description: "IVA",
					Value:       13.0,
				},
			},
			PaymentTypes: []structs.PaymentRequest{
				{
					Code:   "01",
					Amount: 100.0,
				},
			},
		},
		TaxedDiscount:   0,
		IVARetention:    0,
		IncomeRetention: 0,
		TotalIVA:        13.0,
		BalanceInFavor:  0,
	}
}

// CreateInvoiceSummaryWithInvalidTaxCode creates an invoice summary with an invalid tax code
func CreateInvoiceSummaryWithInvalidTaxCode() *structs.InvoiceSummaryRequest {
	summary := CreateDefaultInvoiceSummary()
	summary.Taxes[0].Code = "99"
	return summary
}

// CreateInvoiceSummaryWithMismatchedTotals creates an invoice summary with inconsistent totals
func CreateInvoiceSummaryWithMismatchedTotals() *structs.InvoiceSummaryRequest {
	summary := CreateDefaultInvoiceSummary()
	summary.TotalToPay = 50.0
	return summary
}

// CreateDefaultInvoiceRequest creates a valid default invoice request
func CreateDefaultInvoiceRequest() *structs.CreateInvoiceRequest {
	items := []structs.InvoiceItemRequest{
		CreateDefaultInvoiceItem(0),
		CreateDefaultInvoiceItem(1),
	}

	return &structs.CreateInvoiceRequest{
		Items:     items,
		Receiver:  CreateDefaultReceiver(),
		ModelType: 1,
		Summary:   CreateDefaultInvoiceSummary(),
	}
}

// CreateInvoiceRequestWithInvalidItems creates an invoice request with invalid items
func CreateInvoiceRequestWithInvalidItems() *structs.CreateInvoiceRequest {
	req := CreateDefaultInvoiceRequest()
	req.Items = []structs.InvoiceItemRequest{
		CreateInvoiceItemWithInvalidType(),
	}
	return req
}

// CreateInvoiceRequestWithInvalidSummary creates an invoice request with an invalid summary
func CreateInvoiceRequestWithInvalidSummary() *structs.CreateInvoiceRequest {
	req := CreateDefaultInvoiceRequest()
	req.Summary = CreateInvoiceSummaryWithMismatchedTotals()
	return req
}

// CreateInvoiceRequestWithInvalidReceiver creates an invoice request with an invalid receiver
func CreateInvoiceRequestWithInvalidReceiver() *structs.CreateInvoiceRequest {
	req := CreateDefaultInvoiceRequest()
	req.Receiver = CreateReceiverWithInvalidEmail()
	return req
}

// CreateInvoiceRequestWithAllOptionalFields creates an invoice request with all optional fields
func CreateInvoiceRequestWithAllOptionalFields() *structs.CreateInvoiceRequest {
	req := CreateDefaultInvoiceRequest()
	req.ThirdPartySale = CreateDefaultThirdPartySale()
	req.RelatedDocs = []structs.RelatedDocRequest{CreateDefaultRelatedDocument()}
	req.OtherDocs = []structs.OtherDocRequest{CreateDefaultOtherDocument()}
	req.Appendixes = []structs.AppendixRequest{CreateDefaultAppendix()}
	return req
}

// CreateExpectedInvoiceResponse creates an expected invoice response for testing
func CreateExpectedInvoiceResponse() *respStructs.InvoiceDTEResponse {
	identificacion := &respStructs.DTEIdentification{
		Version:          1,
		Ambiente:         "00",
		TipoDte:          "01",
		NumeroControl:    "DTE-01-00000000-000000000000001",
		CodigoGeneracion: "FF54E9DB-79C3-42CE-B432-EC522C97EFB9",
		TipoModelo:       1,
		TipoOperacion:    1,
		TipoContingencia: nil,
		MotivoContin:     nil,
		FecEmi:           "2025-04-18",
		HorEmi:           "15:30:00",
		TipoMoneda:       "USD",
	}

	emisor := respStructs.DTEIssuer{
		NIT:                 "11111111111111",
		NRC:                 "1111111",
		Nombre:              "EMPRESA DE PRUEBAS SA DE CV",
		CodActividad:        "11111",
		DescActividad:       "Venta al por mayor de otros productos",
		NombreComercial:     utils.ToStringPointer("EJEMPLO SA"),
		TipoEstablecimiento: "02",
		Direccion: respStructs.DTEAddress{
			Departamento: "06",
			Municipio:    "20",
			Complemento:  "BOULEVARD SANTA ELENA SUR, SANTA TECLA",
		},
		Telefono:        "22567890",
		Correo:          "email@gmail.com",
		CodEstableMH:    nil,
		CodEstable:      utils.ToStringPointer("C001"),
		CodPuntoVentaMH: nil,
		CodPuntoVenta:   nil,
	}

	receptor := respStructs.InvoiceReceiver{
		Nombre:        utils.ToStringPointer("Empresa Servicios Generales, S.A. de C.V."),
		TipoDocumento: utils.ToStringPointer("36"),
		NumDocumento:  utils.ToStringPointer("06141804941035"),
		NRC:           utils.ToStringPointer("123456"),
		CodActividad:  utils.ToStringPointer("46900"),
		DescActividad: utils.ToStringPointer("Venta al por mayor de otros productos"),
		Direccion: &respStructs.DTEAddress{
			Departamento: "06",
			Municipio:    "20",
			Complemento:  "Colonia Escalón, Calle La Reforma #123, San Salvador",
		},
		Telefono: utils.ToStringPointer("22123456"),
		Correo:   utils.ToStringPointer("empresa@example.com"),
	}

	items := []respStructs.InvoiceItem{
		{
			NumItem:      1,
			TipoItem:     1,
			Codigo:       utils.ToStringPointer("CODA"),
			Descripcion:  "Producto A",
			Cantidad:     10,
			UniMedida:    59,
			PrecioUni:    5,
			MontoDescu:   0,
			VentaNoSuj:   0,
			VentaExenta:  0,
			VentaGravada: 50,
			Tributos:     []string{"20"},
			PSV:          0,
			NoGravado:    0,
			IvaItem:      6.5,
		},
		{
			NumItem:      2,
			TipoItem:     1,
			Codigo:       utils.ToStringPointer("CODB"),
			Descripcion:  "Producto B",
			Cantidad:     10,
			UniMedida:    59,
			PrecioUni:    5,
			MontoDescu:   0,
			VentaNoSuj:   0,
			VentaExenta:  0,
			VentaGravada: 50,
			Tributos:     []string{"20"},
			PSV:          0,
			NoGravado:    0,
			IvaItem:      6.5,
		},
	}

	resumen := &respStructs.InvoiceSummary{
		TotalNoSuj:          0,
		TotalExenta:         0,
		TotalGravada:        100,
		SubTotalVentas:      100,
		DescuNoSuj:          0,
		DescuExenta:         0,
		DescuGravada:        0,
		PorcentajeDescuento: 0,
		TotalDescu:          0,
		SubTotal:            100,
		ReteRenta:           0,
		IvaRete:             0,
		IvaPerci:            nil,
		MontoTotalOperacion: 100,
		TotalNoGravado:      0,
		TotalPagar:          100,
		TotalLetras:         "CIEN DÓLARES",
		TotalIva:            13.0,
		SaldoFavor:          0,
		CondicionOperacion:  1,
		Pagos: []respStructs.DTEPayment{
			{
				Codigo:     "01",
				MontoPago:  100,
				Referencia: utils.ToStringPointer(""),
				Plazo:      nil,
				Periodo:    nil,
			},
		},
		NumPagoElectronico: nil,
	}

	return &respStructs.InvoiceDTEResponse{
		Identificacion:  identificacion,
		Emisor:          emisor,
		Receptor:        receptor,
		CuerpoDocumento: items,
		Resumen:         resumen,
	}
}
