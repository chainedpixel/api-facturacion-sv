//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// TestE2E_AdversarialLote1 attacks the batch 1 changes against the live API: invalidation replacement rules,
// credit payments with 0.00, item discounts equal to the line amount, unsupported tributes and the issuer scoped listing.
func TestE2E_AdversarialLote1(t *testing.T) {
	client := newAPIClient(t)

	simpleInvoice := func(t *testing.T, desc string, price float64) *emittedDTE {
		t.Helper()
		doc := client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe(desc, 1, price, 0)}}))
		if doc == nil {
			t.Fatal("no se pudo emitir el documento base")
		}
		return doc
	}

	creditInvoice := func(desc string, price float64) *structs.CreateInvoiceRequest {
		return buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{shoe(desc, 2, price, 0)},
			opts: documentOptions{credit: &creditTerms{period: 30, term: "01"}}})
	}

	expectRejectionWith := func(t *testing.T, path string, body interface{}, fragments ...string) {
		t.Helper()
		client.expectRejectedAny(t, path, body, fragments...)
	}

	t.Run("invalidar FE tipo 1 sin reemplazo es rechazado", func(t *testing.T) {
		target := simpleInvoice(t, "FE tipo 1 sin reemplazo", 12.00)
		status, resp, raw := client.post(t, "/dte/invalidation", invalidationRequest(invalidation{target: target, kind: 1, reason: "Error de datos"}))
		if status < 400 || status >= 500 || resp.Success {
			t.Fatalf("se esperaba rechazo 4xx y se obtuvo HTTP %d: %s", status, raw)
		}
		if !containsAny(string(raw), []string{"InvalidReplacementCode1And3", "replacement code is required", "codigo de reemplazo es requerido", "código de reemplazo es requerido"}) {
			t.Errorf("el rechazo no corresponde a InvalidReplacementCode1And3: %s", describeError(resp, raw))
		}
		t.Logf("RECHAZO esperado (HTTP %d): %s", status, describeError(resp, raw))
	})

	t.Run("invalidar FE tipo 2 con reemplazo es rechazado", func(t *testing.T) {
		target := simpleInvoice(t, "FE tipo 2 con reemplazo", 13.00)
		replacement := simpleInvoice(t, "FE de reemplazo indebido", 13.00)
		status, resp, raw := client.post(t, "/dte/invalidation", invalidationRequest(invalidation{target: target, kind: 2, replacement: replacement, reason: "Rescision"}))
		if status < 400 || status >= 500 || resp.Success {
			t.Fatalf("se esperaba rechazo 4xx y se obtuvo HTTP %d: %s", status, raw)
		}
		if !containsAny(string(raw), []string{"InvalidInvalidationType2", "must be null for type 2", "debe ser nulo para el tipo 2"}) {
			t.Errorf("el rechazo no corresponde a InvalidInvalidationType2: %s", describeError(resp, raw))
		}
		t.Logf("RECHAZO esperado (HTTP %d): %s", status, describeError(resp, raw))
	})

	t.Run("FE a credito con dos pagos 0.00 y 0.00 es sellada", func(t *testing.T) {
		req := creditInvoice("Credito con dos pagos en cero", 20.00)
		period, term := 30, "01"
		req.Summary.PaymentTypes = []structs.PaymentRequest{
			{Amount: 0, Period: &period, Term: &term},
			{Amount: 0, Period: &period, Term: &term},
		}
		client.emit(t, "/dte/invoices", req)
	})

	t.Run("FE a credito con pagos 0.00 y monto parcial es rechazada", func(t *testing.T) {
		req := creditInvoice("Credito con pago parcial mezclado", 20.00)
		period, term := 30, "01"
		req.Summary.PaymentTypes = []structs.PaymentRequest{
			{Amount: 0, Period: &period, Term: &term},
			{Amount: 10.00, Period: &period, Term: &term},
		}
		expectRejectionWith(t, "/dte/invoices", req, "InvalidCreditPaymentAmount", "debe ser 0.00", "must be 0.00")
	})

	t.Run("FE a credito con pagos en cero repetidos y un monto parcial al final es rechazada", func(t *testing.T) {
		req := creditInvoice("Credito con tres pagos", 20.00)
		period, term := 30, "01"
		req.Summary.PaymentTypes = []structs.PaymentRequest{
			{Amount: 0, Period: &period, Term: &term},
			{Amount: 0, Period: &period, Term: &term},
			{Amount: 0.01, Period: &period, Term: &term},
		}
		expectRejectionWith(t, "/dte/invoices", req, "InvalidCreditPaymentAmount", "debe ser 0.00", "must be 0.00")
	})

	t.Run("FE al contado con pago 0.00 es rechazada", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{shoe("Contado con pago en cero", 1, 15.00, 0)}})
		req.Summary.PaymentTypes[0].Amount = 0
		expectRejectionWith(t, "/dte/invoices", req, "InvalidPaymentTotal", "no coincide")
	})

	t.Run("FE con pago de monto negativo es rechazada", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{shoe("Contado con pago negativo", 1, 15.00, 0)}})
		req.Summary.PaymentTypes[0].Amount = -15.00
		expectRejectionWith(t, "/dte/invoices", req)
	})

	t.Run("FE con descuento de item igual al monto de la linea (reporte de comportamiento)", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{
			shoe("Articulo regalado", 1, 20.00, 20.00), shoe("Articulo normal", 1, 10.00, 0),
		}})
		status, resp, raw := client.post(t, "/dte/invoices", req)
		stamped := resp.ReceptionStamp != nil && *resp.ReceptionStamp != ""
		t.Logf("REPORTE descuento == monto de linea: HTTP %d | sellado=%v | %s", status, stamped, describeError(resp, raw))
		if status >= 500 {
			t.Errorf("error interno ante descuento igual al monto de la linea: %s", raw)
		}
	})

	t.Run("FE con descuento de item superior al monto de la linea es rechazada", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{shoe("Descuento excesivo", 1, 20.00, 20.00), shoe("Otro", 1, 10.00, 0)}})
		req.Items[0].Discount = 25.00
		expectRejectionWith(t, "/dte/invoices", req)
	})

	t.Run("CCF con tributo de resumen D1 es rechazado", func(t *testing.T) {
		req := validCCF()
		req.Summary.Taxes = append(req.Summary.Taxes, structs.TaxRequest{Code: "D1", Description: "FOVIAL", Value: 0.20})
		expectRejectionWith(t, "/dte/ccf", req, "UnsupportedTaxCode", "Unsupported tax code", "no soportado")
	})

	t.Run("CCF con tributo de resumen C8 es rechazado", func(t *testing.T) {
		req := validCCF()
		req.Summary.Taxes = append(req.Summary.Taxes, structs.TaxRequest{Code: "C8", Description: "COTRANS", Value: 0.10})
		expectRejectionWith(t, "/dte/ccf", req, "UnsupportedTaxCode", "Unsupported tax code", "no soportado")
	})

	t.Run("CCF con tributo de resumen d1 en minuscula es rechazado", func(t *testing.T) {
		req := validCCF()
		req.Summary.Taxes = append(req.Summary.Taxes, structs.TaxRequest{Code: "d1", Description: "FOVIAL", Value: 0.20})
		expectRejectionWith(t, "/dte/ccf", req, "InvalidTaxType", "UnsupportedTaxCode", "tax type", "tipo de impuesto")
	})

	t.Run("CCF con D1 en los tributos de un item es rechazado", func(t *testing.T) {
		req := validCCF()
		req.Items[0].Taxes = append(req.Items[0].Taxes, "D1")
		expectRejectionWith(t, "/dte/ccf", req, "UnsupportedTaxCode", "Unsupported tax code", "no soportado")
	})

	t.Run("FSE con item de tipo impuesto es rechazado", func(t *testing.T) {
		req := validFSE()
		req.Items[0].Type = 4
		expectRejectionWith(t, "/dte/fse", req, "InvalidFSEItemType", "tax items (type 4)", "tipo impuesto")
	})

	t.Run("FE con otro documento asociado codigo 4 es rechazada", func(t *testing.T) {
		req := validInvoice()
		req.OtherDocs = []structs.OtherDocRequest{{DocumentCode: 4, Description: str("Transporte"), Detail: str("Detalle")}}
		expectRejectionWith(t, "/dte/invoices", req, "InvalidAssociatedDocumentCodeForType", "is not allowed for document type", "no está permitido", "no esta permitido")
	})

	t.Run("GET /dte?all=true devuelve solo documentos del NIT del emisor", func(t *testing.T) {
		status, raw := client.getJSON(t, "/dte?all=true&page_size=100")
		if status != http.StatusOK {
			t.Fatalf("GET /dte?all=true devolvio HTTP %d: %s", status, raw)
		}
		var resp struct {
			Data struct {
				Documents []struct {
					Document json.RawMessage `json:"document"`
				} `json:"documents"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("respuesta ilegible: %v", err)
		}
		issuers := map[string]int{}
		for _, doc := range resp.Data.Documents {
			var parsed struct {
				Emisor struct {
					NIT string `json:"nit"`
				} `json:"emisor"`
			}
			if err := json.Unmarshal(doc.Document, &parsed); err == nil && parsed.Emisor.NIT != "" {
				issuers[parsed.Emisor.NIT]++
			}
		}
		t.Logf("documentos devueltos %d | emisores distintos %v", len(resp.Data.Documents), issuers)
		if len(resp.Data.Documents) == 0 {
			t.Errorf("la consulta all=true no devolvio documentos tras emitir varios")
			return
		}
		if len(issuers) == 0 {
			t.Logf("la respuesta no expone emisor.nit en document; se verifica solo HTTP 200")
			return
		}
		if len(issuers) != 1 {
			t.Errorf("all=true devolvio documentos de mas de un emisor: %v", issuers)
		}
		expectedNIT := strings.ReplaceAll(os.Getenv("E2E_MH_USER"), "-", "")
		for nit := range issuers {
			if expectedNIT != "" && strings.ReplaceAll(nit, "-", "") != expectedNIT {
				t.Logf("el NIT del emisor %s difiere de E2E_MH_USER %s (puede ser otro NIT de pruebas)", nit, expectedNIT)
			}
		}
	})

	t.Run("GET /dte?all=true con parametros hostiles no rompe la consulta", func(t *testing.T) {
		for _, query := range []string{"all=true&nit=%27%20OR%201%3D1%20--", "all=%27%3B%20DROP%20TABLE%20dte_documents%3B--", "all=TRUE"} {
			status, raw := client.getJSON(t, "/dte?"+query)
			if status != http.StatusOK {
				t.Errorf("GET /dte?%s devolvio HTTP %d: %s", query, status, raw)
			}
		}
	})

	t.Run("GET /dte/{uuid desconocido} devuelve el error esperado", func(t *testing.T) {
		for _, id := range []string{"00000000-0000-4000-8000-000000000000", "NO-ES-UN-UUID", "' OR '1'='1"} {
			path := "/dte/" + strings.ReplaceAll(id, " ", "%20")
			status, raw := client.getJSON(t, path)
			if status != http.StatusBadRequest && status != http.StatusNotFound {
				t.Errorf("GET %s devolvio HTTP %d, se esperaba 400 o 404: %s", path, status, raw)
				continue
			}
			var resp apiResponse
			_ = json.Unmarshal(raw, &resp)
			if resp.Success {
				t.Errorf("GET %s no debe responder success=true: %s", path, raw)
			}
			t.Logf("GET %s -> HTTP %d | %s", id, status, describeError(resp, raw))
		}
	})
}
