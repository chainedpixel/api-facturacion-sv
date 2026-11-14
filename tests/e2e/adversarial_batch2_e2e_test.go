//go:build e2e

package e2e

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// TestE2E_AdversarialLote2 ataca las correcciones del lote 2: documentos relacionados de la FE, impuestos de la ND
// y saldo inicial del CCF neto de descuentos globales.
func TestE2E_AdversarialLote2(t *testing.T) {
	client := newAPIClient(t)

	var remission *emittedDTE
	t.Run("NR previa para relacionar desde la FE", func(t *testing.T) {
		remission = client.emit(t, "/dte/remissionnote", remissionFor(remissionReceiver()))
	})

	feWithRelated := func(related []structs.RelatedDocRequest) *structs.CreateInvoiceRequest {
		return buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta de mercaderia remitida", 2, 25.00, 0), shoe("Venta de calzado remitido", 1, 30.00, 0)},
			opts:  documentOptions{related: related}})
	}

	t.Run("FE con documento relacionado 04 e items vinculados obtiene sello", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		client.emit(t, "/dte/invoices", feWithRelated(relatedRemission(remission)))
	})

	t.Run("FE con documento relacionado tipo 03 se rechaza", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		related := relatedRemission(remission)
		related[0].DocumentType = "03"
		client.expectRejected(t, "/dte/invoices", feWithRelated(related), "InvalidRelatedDocDTEType")
	})

	t.Run("FE con related_docs y ningun item vinculado se rechaza", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		req := feWithRelated(relatedRemission(remission))
		for i := range req.Items {
			req.Items[i].RelatedDoc = nil
		}
		client.expectRejected(t, "/dte/invoices", req, "MissingItemRelatedDoc")
	})

	t.Run("FE con item que referencia un documento no declarado se rechaza", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		req := feWithRelated(relatedRemission(remission))
		req.Items[1].RelatedDoc = str("00000000-0000-0000-0000-000000000000")
		client.expectRejected(t, "/dte/invoices", req, "InvalidItemRelatedDoc")
	})

	var base *emittedDTE
	t.Run("CCF de 100.00 con descuento global del 10 por ciento", func(t *testing.T) {
		base = client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items:  []itemSpec{bread("Pan dulce mayoreo", 1, 100.00, 0)},
			global: globalDiscounts{taxed: 10.00}}))
	})

	t.Run("NC de 95.00 excede el saldo neto y se rechaza", func(t *testing.T) {
		if base == nil {
			t.Skip("el CCF base no obtuvo sello")
		}
		client.expectRejected(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
			items: []itemSpec{bread("Devolucion excedida", 1, 95.00, 0)}}), "InvalidCreditNoteTransaction")
	})

	t.Run("NC de 90.00 iguala el saldo neto y obtiene sello", func(t *testing.T) {
		if base == nil {
			t.Skip("el CCF base no obtuvo sello")
		}
		client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
			items: []itemSpec{bread("Devolucion total neta", 1, 90.00, 0)}}))
	})

	t.Run("ND con IVA errado por 1.00 se rechaza", func(t *testing.T) {
		if base == nil {
			t.Skip("el CCF base no obtuvo sello")
		}
		req := buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
			items: []itemSpec{bread("Cargo adicional", 1, 10.00, 0)}})
		req.Summary.Taxes[0].Value += 1.00
		req.Summary.TotalOperation += 1.00
		req.Summary.TotalToPay += 1.00
		req.Summary.PaymentTypes[0].Amount = req.Summary.TotalToPay
		client.expectRejected(t, "/dte/debitnote", req, "InvalidTaxCalculation")
	})
}
