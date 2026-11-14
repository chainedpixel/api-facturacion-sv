//go:build e2e

package e2e

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// TestE2E_CreditoYDocumentosRelacionados covers the credit sale payment contract (null code, 0.00 amount, null
// reference, term and period) and the strict link between items and related documents in FE and CCF.
func TestE2E_CreditoYDocumentosRelacionados(t *testing.T) {
	client := newAPIClient(t)

	credit := documentOptions{credit: &creditTerms{period: 30, term: "01"}}
	creditInvoice := func(desc string) *structs.CreateInvoiceRequest {
		return buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe(desc, 2, 25.00, 0)}, opts: credit})
	}

	t.Run("FE a credito con codigo null, monto 0.00, referencia null, plazo y periodo obtiene sello", func(t *testing.T) {
		client.emit(t, "/dte/invoices", creditInvoice("Venta a credito contrato nuevo"))
	})

	t.Run("CCF a credito con codigo null, monto 0.00, referencia null, plazo y periodo obtiene sello", func(t *testing.T) {
		client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Pedido a credito contrato nuevo", 2, 25.00, 0)}, opts: credit}))
	})

	t.Run("venta a credito con codigo de forma de pago 03 se rechaza localmente", func(t *testing.T) {
		req := creditInvoice("Credito con codigo de pago")
		req.Summary.PaymentTypes[0].Code = "03"
		client.expectRejected(t, "/dte/invoices", req, "CreditPaymentCodeNotAllowed")
	})

	t.Run("CCF a credito con codigo de forma de pago 03 se rechaza localmente", func(t *testing.T) {
		req := buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("CCF credito con codigo de pago", 2, 25.00, 0)}, opts: credit})
		req.Summary.PaymentTypes[0].Code = "03"
		client.expectRejected(t, "/dte/ccf", req, "CreditPaymentCodeNotAllowed")
	})

	t.Run("venta a credito sin plazo se rechaza localmente", func(t *testing.T) {
		req := creditInvoice("Credito sin plazo")
		req.Summary.PaymentTypes[0].Term = nil
		client.expectRejected(t, "/dte/invoices", req, "CreditPaymentTermRequired")
	})

	t.Run("venta a credito sin periodo se rechaza localmente", func(t *testing.T) {
		req := creditInvoice("Credito sin periodo")
		req.Summary.PaymentTypes[0].Period = nil
		client.expectRejected(t, "/dte/invoices", req, "CreditPaymentPeriodRequired")
	})

	t.Run("venta a credito con referencia se rechaza localmente", func(t *testing.T) {
		req := creditInvoice("Credito con referencia")
		req.Summary.PaymentTypes[0].Reference = str("REF-CREDITO")
		client.expectRejected(t, "/dte/invoices", req, "CreditPaymentReferenceNotAllowed")
	})

	t.Run("venta al contado sin codigo de forma de pago se rechaza localmente", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{shoe("Contado sin codigo", 1, 15.00, 0)}})
		req.Summary.PaymentTypes[0].Code = ""
		client.expectRejected(t, "/dte/invoices", req, "PaymentCodeRequired")
	})

	var remission *emittedDTE
	t.Run("NR previa para relacionar", func(t *testing.T) {
		remission = client.emit(t, "/dte/remissionnote", remissionFor(remissionReceiver()))
	})

	feWithRelated := func(desc string) *structs.CreateInvoiceRequest {
		return buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe(desc, 2, 25.00, 0), shoe("Articulo adicional no remitido", 1, 30.00, 0)},
			opts:  documentOptions{related: relatedRemission(remission)}})
	}

	t.Run("FE que relaciona una NR con todos los items vinculados obtiene sello", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		client.emit(t, "/dte/invoices", feWithRelated("Mercaderia remitida"))
	})

	t.Run("FE que relaciona una NR con un item vinculado y otro item en null se rechaza localmente", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		req := feWithRelated("Mercaderia remitida con item suelto")
		req.Items[1].RelatedDoc = nil
		client.expectRejected(t, "/dte/invoices", req, "MissingItemRelatedDoc")
	})

	t.Run("FE que relaciona una NR sin que ningun item la referencie se rechaza localmente", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		req := feWithRelated("Mercaderia sin vinculo")
		for i := range req.Items {
			req.Items[i].RelatedDoc = nil
		}
		client.expectRejected(t, "/dte/invoices", req, "MissingItemRelatedDoc")
	})

	t.Run("FE sin documentos relacionados con un item que referencia un documento se rechaza localmente", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{shoe("Item con referencia huerfana", 1, 20.00, 0)}})
		req.Items[0].RelatedDoc = str(remission.GenerationCode)
		client.expectRejected(t, "/dte/invoices", req, "InvalidItemRelatedDoc")
	})

	t.Run("CCF que relaciona una NR con un item vinculado y otro item en null se rechaza localmente", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		req := buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Mercaderia remitida CCF", 2, 25.00, 0), shoe("Articulo CCF no remitido", 1, 30.00, 0)},
			opts:  documentOptions{related: relatedRemission(remission)}})
		req.Items[1].RelatedDoc = nil
		client.expectRejected(t, "/dte/ccf", req, "MissingItemRelatedDoc")
	})
}
