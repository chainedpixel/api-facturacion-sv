//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// Rechazos esperados por HTTP: cada caso parte de un documento valido de la calculadora, introduce
// un unico error de la normativa y exige que la API lo rechace antes de transmitir a Hacienda.

func validInvoice() *structs.CreateInvoiceRequest {
	return buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{
		shoe("Zapato casual", 2, 25.00, 0), shoe("Zapato deportivo", 1, 30.00, 3.00),
	}})
}

func validCCF() *structs.CreateCreditFiscalRequest {
	return buildCCF(ccfSpec{receiver: ccfReceiver(), items: []itemSpec{
		bread("Pan dulce caja", 3, 20.00, 0), bread("Pastel mediano", 1, 40.00, 5.00),
	}})
}

func validFSE() *structs.CreateFSERequest {
	return buildFSE(fseSpec{receiver: excludedSubject(), incomeRetention: 0.10,
		items: []itemSpec{{description: "Servicio de limpieza", itemType: 2, quantity: 4, unitPrice: 10.00}}})
}

func TestE2E_RechazosEsperados(t *testing.T) {
	client := newAPIClient(t)

	cases := []struct {
		name     string
		path     string
		body     func() interface{}
		wantCode string
	}{
		// ---- FE ----
		{"FE con retencion de renta (eliminada en v2.0)", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Summary.IncomeRetention = 7.70
			req.Summary.TotalToPay -= 7.70
			req.Summary.PaymentTypes[0].Amount = req.Summary.TotalToPay
			return req
		}, "IncomeRetentionNotAllowed"},
		{"FE con tributo 20 en un item", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Items[0].Type = 2
			req.Items[0].Taxes = []string{"20"}
			return req
		}, "TaxNotAllowedInInvoice"},
		{"FE con totalIva calculado sin dividir entre 1.13", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Summary.TotalIVA = 10.01
			return req
		}, "InvalidTotalIVA"},
		{"FE con ivaItem que no corresponde a la venta", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Items[0].IVAItem = 6.50
			return req
		}, "InvalidIVAItemCalculation"},
		{"FE con venta por item distinta de precio x cantidad - descuento", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Items[1].TaxedSale = 30.00
			req.Items[1].IVAItem = 3.45132743 // ivaItem coherente con la venta declarada: se aisla la formula de venta
			return req
		}, "InvalidItemSaleAmount"},
		{"FE con subTotalVentas mal sumado", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Summary.SubTotalSales = 70.00
			return req
		}, "InvalidSubTotalSales"},
		{"FE con suma de pagos distinta del total", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Summary.PaymentTypes[0].Amount = 50.00
			return req
		}, ""},
		{"FE con total a pagar de 3 decimales", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Summary.TotalToPay = 77.005
			return req
		}, ""},
		{"FE con venta por cuenta de terceros sin domicilio fiscal", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.ThirdPartySale = &structs.ThirdPartySaleRequest{NIT: "07023108921027", Name: "Tercero sin domicilio"}
			return req
		}, "FiscalDomicile"},
		{"FE con tipo de documento del receptor sin numero", "/dte/invoices", func() interface{} {
			req := validInvoice()
			req.Receiver.DocumentNumber = nil
			return req
		}, ""},

		// ---- CCF ----
		{"CCF con monto total sin el IVA", "/dte/ccf", func() interface{} {
			req := validCCF()
			req.Summary.TotalOperation = 95.00
			req.Summary.TotalToPay = 95.00
			req.Summary.PaymentTypes[0].Amount = 95.00
			return req
		}, "InvalidTotalOperation"},
		{"CCF con IVA del resumen mal calculado", "/dte/ccf", func() interface{} {
			req := validCCF()
			req.Summary.Taxes[0].Value = 13.00
			req.Summary.TotalOperation = 108.00
			req.Summary.TotalToPay = 108.00
			req.Summary.PaymentTypes[0].Amount = 108.00
			return req
		}, "InvalidTaxCalculation"},
		{"CCF con percepcion que no es el 1%", "/dte/ccf", func() interface{} {
			req := validCCF()
			req.Summary.IVAPerception = 5.00
			req.Summary.TotalToPay += 5.00
			req.Summary.PaymentTypes[0].Amount = req.Summary.TotalToPay
			return req
		}, "InvalidPerceptionAmount"},
		{"CCF con retencion de renta (eliminada en v2.0)", "/dte/ccf", func() interface{} {
			req := validCCF()
			req.Summary.IncomeRetention = 9.50
			return req
		}, "IncomeRetentionNotAllowed"},
		{"CCF con venta por item mal calculada", "/dte/ccf", func() interface{} {
			req := validCCF()
			req.Items[1].TaxedSale = 40.00
			return req
		}, "InvalidItemSaleAmount"},
		{"CCF con item gravado sin tributo 20", "/dte/ccf", func() interface{} {
			req := validCCF()
			req.Items[0].Taxes = nil
			return req
		}, ""},
		{"CCF sin NIT del receptor", "/dte/ccf", func() interface{} {
			req := validCCF()
			req.Receiver.NIT = nil
			return req
		}, ""},

		// ---- FSE ----
		{"FSE con retencion de IVA (eliminada en v2.0)", "/dte/fse", func() interface{} {
			req := validFSE()
			req.Summary.IVARetention = 0.40
			return req
		}, "IVARetentionNotAllowedInFSE"},
		{"FSE con total a pagar distinto de subtotal - renta", "/dte/fse", func() interface{} {
			req := validFSE()
			req.Summary.TotalToPay = 40.00
			req.Summary.PaymentTypes[0].Amount = 40.00
			return req
		}, "FSETaxInvalidTotalToPay"},
		{"FSE sin direccion del sujeto excluido", "/dte/fse", func() interface{} {
			req := validFSE()
			req.Receiver.Address = nil
			return req
		}, "RequiredField"},
		{"FSE con compra mal calculada", "/dte/fse", func() interface{} {
			req := validFSE()
			req.Items[0].Purchase = 45.00
			return req
		}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client.expectRejected(t, tc.path, tc.body(), tc.wantCode)
		})
	}

	t.Run("NC sobre un CCF", func(t *testing.T) {
		base := emitBase(t, client, ccfSpec{receiver: ccfReceiver(), items: []itemSpec{shoe("Venta base", 5, 40.00, 0)}})
		validNote := func() *structs.CreateCreditNoteRequest {
			return buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{shoe("Devolucion", 1, 40.00, 0)}})
		}
		t.Run("NC con retencion de renta", func(t *testing.T) {
			req := validNote()
			req.Summary.IncomeRetention = 4.00
			client.expectRejected(t, "/dte/creditnote", req, "IncomeRetentionNotAllowed")
		})
		t.Run("NC con total a pagar mal calculado", func(t *testing.T) {
			req := validNote()
			req.Summary.TotalToPay += 3.00
			client.expectRejected(t, "/dte/creditnote", req, "InvalidTotalToPayCalculation")
		})
		t.Run("NC con monto total con formula anterior (sin tributos)", func(t *testing.T) {
			req := validNote()
			req.Summary.TotalOperation = 40.00
			client.expectRejected(t, "/dte/creditnote", req, "InvalidTotalOperation")
		})
		t.Run("NC con descuento global", func(t *testing.T) {
			req := validNote()
			req.Summary.TaxedDiscount = 5.00
			client.expectRejected(t, "/dte/creditnote", req, "GlobalDiscountNotAllowedInAdjustment")
		})
		t.Run("NC sobre un CCF que no existe", func(t *testing.T) {
			req := validNote()
			req.RelatedDocs[0].DocumentNumber = "00000000-0000-4000-8000-000000000000"
			for i := range req.Items {
				req.Items[i].RelatedDoc = str("00000000-0000-4000-8000-000000000000")
			}
			client.expectRejected(t, "/dte/creditnote", req, "")
		})
	})
}

// Limites de campos que la normativa v2.0 amplio y que deben aceptarse
func TestE2E_LimitesNormativa(t *testing.T) {
	client := newAPIClient(t)

	t.Run("descripcion de item de 1500 caracteres (ampliada en v2.0)", func(t *testing.T) {
		description := strings.Repeat("Zapato de cuero genuino con suela antideslizante. ", 30)[:1500]
		client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe(description, 1, 50.00, 0)}}))
	})
}
