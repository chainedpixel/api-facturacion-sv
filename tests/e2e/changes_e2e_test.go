//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestE2E_CambiosNormativa covers credit sales paid with 0.00, item discounts above 100.00 and the
// processing date returned by the consult endpoint.
func TestE2E_CambiosNormativa(t *testing.T) {
	client := newAPIClient(t)

	zeroCredit := documentOptions{credit: &creditTerms{period: 30, term: "01"}}

	t.Run("FE a credito con pago de 0.00", func(t *testing.T) {
		client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta a credito sin abono", 2, 25.00, 0)}, opts: zeroCredit}))
	})

	t.Run("CCF a credito con pago de 0.00", func(t *testing.T) {
		client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Pedido a credito sin abono", 2, 25.00, 0)}, opts: zeroCredit}))
	})

	t.Run("FE a credito con pago igual al total es rechazada", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Credito con pago igual al total", 2, 25.00, 0)}, opts: zeroCredit})
		req.Summary.PaymentTypes[0].Amount = req.Summary.TotalToPay
		client.expectRejectedAny(t, "/dte/invoices", req, "InvalidCreditPaymentAmount", "debe ser 0.00", "must be 0.00")
	})

	t.Run("FE con descuento de item mayor a 100.00", func(t *testing.T) {
		client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Articulo con descuento alto", 1, 500.00, 150.00)}}))
	})

	t.Run("consulta devuelve processing_date tras la emision", func(t *testing.T) {
		doc := client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta para consulta", 1, 20.00, 0)}}))
		if doc == nil {
			t.Fatal("no se pudo emitir la FE")
		}
		status, raw := client.getJSON(t, "/dte/"+doc.GenerationCode)
		if status != http.StatusOK {
			t.Fatalf("GET /dte/{codigo} devolvio HTTP %d: %s", status, raw)
		}
		var resp struct {
			Data struct {
				ProcessingDate string `json:"processing_date"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("respuesta de consulta ilegible: %v", err)
		}
		if resp.Data.ProcessingDate == "" {
			t.Errorf("processing_date vacio en la consulta de %s: %s", doc.GenerationCode, raw)
			return
		}
		t.Logf("processing_date %s | %s", resp.Data.ProcessingDate, doc.GenerationCode)
	})
}
