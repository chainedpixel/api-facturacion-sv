//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// Exploratory probe of the tourism taxes (59 and 71) that were never confirmed against Hacienda. It is skipped
// unless E2E_TOURISM_PROBE=1. Hacienda outcomes are only logged, never failed: each case prints one PROBE line with
// the HTTP status, the stamp, the observations and the API error, followed by a SENT line with the exact taxes sent.
// Documents are not invalidated by the probe.

// probeWire is the JSON shape of a request needed to report what was sent.
type probeWire struct {
	Items []struct {
		Type      int      `json:"type"`
		TaxedSale float64  `json:"taxed_sale"`
		Taxes     []string `json:"taxes"`
	} `json:"items"`
	Summary struct {
		Taxes []struct {
			Code  string  `json:"code"`
			Value float64 `json:"value"`
		} `json:"taxes"`
		TotalTaxed     float64 `json:"total_taxed"`
		TaxedDiscount  float64 `json:"taxed_discount"`
		IVAPerception  float64 `json:"iva_perception"`
		SubTotal       float64 `json:"sub_total"`
		TotalOperation float64 `json:"total_operation"`
		TotalToPay     float64 `json:"total_to_pay"`
	} `json:"summary"`
}

// describeSent renders the item taxes and the summary taxes and totals of a request body.
func describeSent(t *testing.T, body interface{}) string {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("no se pudo serializar el request de la sonda: %v", err)
	}
	var wire probeWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("no se pudo leer el request de la sonda: %v", err)
	}
	var items, taxes []string
	for i, item := range wire.Items {
		items = append(items, fmt.Sprintf("#%d{type=%d taxed_sale=%.2f taxes=%v}", i+1, item.Type, item.TaxedSale, item.Taxes))
	}
	for _, tax := range wire.Summary.Taxes {
		taxes = append(taxes, fmt.Sprintf("%s=%.2f", tax.Code, tax.Value))
	}
	s := wire.Summary
	return fmt.Sprintf("items=[%s] summary_taxes=[%s] total_taxed=%.2f taxed_discount=%.2f iva_perception=%.2f sub_total=%.2f total_operation=%.2f total_to_pay=%.2f",
		strings.Join(items, " "), strings.Join(taxes, " "), s.TotalTaxed, s.TaxedDiscount, s.IVAPerception, s.SubTotal, s.TotalOperation, s.TotalToPay)
}

// tourismService is a taxed service item (type 2): the CCF item rules only allow a tax other than 20 on services.
func tourismService(description string, price float64) itemSpec {
	item := shoe(description, 1, price, 0)
	item.itemType = 2
	return item
}

// airportTax is the departure tourism tax 71, a fixed amount per document.
func airportTax() structs.TaxRequest {
	return structs.TaxRequest{Code: "71", Description: "Turismo: salida del pais por via aerea", Value: 7.00}
}

// tourismOnCarrying is the tax 59 for documents without IVA in the price: 5% of the taxed sale of the items that
// carry the tax, as Hacienda computes it.
func tourismOnCarrying(carryingTaxed float64) structs.TaxRequest {
	tax := tourismTax()
	return structs.TaxRequest{Code: tax.code, Description: tax.description, Value: r2(d(carryingTaxed).Mul(d(tax.rate)))}
}

// addProbeTaxes appends item tax codes (itemCodes[i] goes to item i) and the given summary taxes, then recomputes
// total_operation, total_to_pay, the payments and the amount in words. dteType is the payment shape key of payments().
func addProbeTaxes(items []*structs.ItemRequest, itemCodes [][]string, summary *structs.SummaryRequest, dteType string, added ...structs.TaxRequest) {
	for i, codes := range itemCodes {
		items[i].Taxes = append(items[i].Taxes, codes...)
	}
	extra := d(0)
	for _, tax := range added {
		summary.Taxes = append(summary.Taxes, tax)
		extra = extra.Add(d(tax.Value))
	}
	summary.TotalOperation = r2(d(summary.TotalOperation).Add(extra))
	summary.TotalToPay = r2(d(summary.TotalToPay).Add(extra))
	summary.PaymentTypes = payments(dteType, summary.TotalToPay, documentOptions{})
	summary.TotalInWords = str(amountInWords(summary.TotalToPay))
}

func invoiceItemPointers(req *structs.CreateInvoiceRequest) []*structs.ItemRequest {
	out := make([]*structs.ItemRequest, len(req.Items))
	for i := range req.Items {
		out[i] = &req.Items[i].ItemRequest
	}
	return out
}

func ccfItemPointers(req *structs.CreateCreditFiscalRequest) []*structs.ItemRequest {
	out := make([]*structs.ItemRequest, len(req.Items))
	for i := range req.Items {
		out[i] = &req.Items[i].ItemRequest
	}
	return out
}

func creditNoteItemPointers(req *structs.CreateCreditNoteRequest) []*structs.ItemRequest {
	out := make([]*structs.ItemRequest, len(req.Items))
	for i := range req.Items {
		out[i] = &req.Items[i].ItemRequest
	}
	return out
}

func debitNoteItemPointers(req *structs.CreateDebitNoteRequest) []*structs.ItemRequest {
	out := make([]*structs.ItemRequest, len(req.Items))
	for i := range req.Items {
		out[i] = &req.Items[i].ItemRequest
	}
	return out
}

// probeEmit posts the document and logs the PROBE and SENT lines without failing on any API or Hacienda outcome.
// It returns the emitted document when Hacienda granted a stamp, nil otherwise.
func (c *apiClient) probeEmit(t *testing.T, id, path string, body interface{}) *emittedDTE {
	t.Helper()
	sent := describeSent(t, body)
	status, resp, raw := c.post(t, path, body)

	stamp := resp.ReceptionStamp != nil && *resp.ReceptionStamp != ""
	var document struct {
		Identificacion struct {
			CodigoGeneracion string `json:"codigoGeneracion"`
			NumeroControl    string `json:"numeroControl"`
			FecEmi           string `json:"fecEmi"`
		} `json:"identificacion"`
	}
	_ = json.Unmarshal(resp.Data, &document)

	estado := "RECHAZADO_O_SIN_SELLO"
	if stamp {
		estado = "PROCESADO"
	}
	observations := string(resp.Observations)
	if observations == "" {
		observations = "[]"
	}
	apiError := "none"
	if resp.Error != nil {
		apiError = describeError(resp, raw)
	} else if !stamp {
		apiError = string(raw)
	}
	yes := "no"
	if stamp {
		yes = "yes"
	}
	t.Logf("PROBE %s | http=%d | estado=%s | codigoGeneracion=%s | selloRecibido=%s | observaciones=%s | error=%s",
		id, status, estado, document.Identificacion.CodigoGeneracion, yes, observations, apiError)
	t.Logf("SENT %s | %s", id, sent)

	if !stamp {
		return nil
	}
	return &emittedDTE{ReceptionStamp: *resp.ReceptionStamp, GenerationCode: document.Identificacion.CodigoGeneracion,
		ControlNumber: document.Identificacion.NumeroControl, EmissionDate: document.Identificacion.FecEmi}
}

// mixedTourismCCF builds a CCF whose first item carries 59 and whose other items carry only the IVA.
func mixedTourismCCF(items ...itemSpec) *structs.CreateCreditFiscalRequest {
	req := buildCCF(ccfSpec{receiver: ccfReceiver(), items: items})
	addProbeTaxes(ccfItemPointers(req), [][]string{{"59"}}, &req.Summary.SummaryRequest, "03",
		tourismOnCarrying(req.Items[0].TaxedSale))
	return req
}

// allTourismCCF builds a CCF where every item carries 59.
func allTourismCCF(items ...itemSpec) *structs.CreateCreditFiscalRequest {
	req := buildCCF(ccfSpec{receiver: ccfReceiver(), items: items})
	codes := make([][]string, len(items))
	for i := range codes {
		codes[i] = []string{"59"}
	}
	addProbeTaxes(ccfItemPointers(req), codes, &req.Summary.SummaryRequest, "03", tourismOnCarrying(req.Summary.TotalTaxed))
	return req
}

// TestE2E_SondeoTributosTurismo probes how Hacienda treats the tourism taxes 59 and 71 in the cases never tested.
func TestE2E_SondeoTributosTurismo(t *testing.T) {
	if os.Getenv("E2E_TOURISM_PROBE") != "1" {
		t.Skip("define E2E_TOURISM_PROBE=1 para ejecutar la sonda de tributos de turismo")
	}
	client := newAPIClient(t)
	t.Logf("LOCALE los mensajes usan el idioma global del servidor (config/translations.go); el cliente E2E no envia Accept-Language")

	var mixedCCF *emittedDTE
	t.Run("T59-CCF-MIXED", func(t *testing.T) {
		mixedCCF = client.probeEmit(t, "T59-CCF-MIXED", "/dte/ccf", mixedTourismCCF(
			tourismService("Servicio turistico gravado", 100.00), shoe("Zapato casual sin tributo", 1, 50.00, 0)))
	})

	t.Run("T59-CCF-ALL", func(t *testing.T) {
		client.probeEmit(t, "T59-CCF-ALL", "/dte/ccf", allTourismCCF(
			tourismService("Servicio turistico gravado A", 100.00), tourismService("Servicio turistico gravado B", 50.00)))
	})

	t.Run("T59-NC-MIXED", func(t *testing.T) {
		if mixedCCF == nil {
			t.Skip("T59-CCF-MIXED no obtuvo sello")
		}
		req := buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: mixedCCF, items: []itemSpec{
			tourismService("Servicio turistico gravado", 40.00), shoe("Zapato casual sin tributo", 1, 20.00, 0)}})
		addProbeTaxes(creditNoteItemPointers(req), [][]string{{"59"}}, &req.Summary.SummaryRequest, "",
			tourismOnCarrying(req.Items[0].TaxedSale))
		client.probeEmit(t, "T59-NC-MIXED", "/dte/creditnote", req)
	})

	t.Run("T59-ND-MIXED", func(t *testing.T) {
		if mixedCCF == nil {
			t.Skip("T59-CCF-MIXED no obtuvo sello")
		}
		req := buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: mixedCCF, items: []itemSpec{
			tourismService("Servicio turistico gravado", 40.00), shoe("Zapato casual sin tributo", 1, 20.00, 0)}})
		addProbeTaxes(debitNoteItemPointers(req), [][]string{{"59"}}, &req.Summary.SummaryRequest, "",
			tourismOnCarrying(req.Items[0].TaxedSale))
		client.probeEmit(t, "T59-ND-MIXED", "/dte/debitnote", req)
	})

	t.Run("T71-FE", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Servicio de salida del pais", 1, 100.00, 0)}})
		addProbeTaxes(invoiceItemPointers(req), [][]string{{"71"}}, &req.Summary.SummaryRequest, "01", airportTax())
		client.probeEmit(t, "T71-FE", "/dte/invoices", req)
	})

	t.Run("T71-CCF", func(t *testing.T) {
		req := buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{tourismService("Servicio de salida del pais", 100.00)}})
		addProbeTaxes(ccfItemPointers(req), [][]string{{"71"}}, &req.Summary.SummaryRequest, "03", airportTax())
		client.probeEmit(t, "T71-CCF", "/dte/ccf", req)
	})

	t.Run("T59-71-FE-BOTH", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(), extraTax: tourismTax(),
			items: []itemSpec{shoe("Servicio turistico y salida del pais", 1, 100.00, 0)}})
		addProbeTaxes(invoiceItemPointers(req), [][]string{{"71"}}, &req.Summary.SummaryRequest, "01", airportTax())
		client.probeEmit(t, "T59-71-FE-BOTH", "/dte/invoices", req)
	})

	t.Run("MSG-PERCEPTION", func(t *testing.T) {
		req := buildCCF(ccfSpec{receiver: ccfReceiver(), ivaPerception: true,
			items:  []itemSpec{shoe("Venta a pequeno contribuyente", 1, 700.00, 0)},
			global: globalDiscounts{taxed: 50.00}})
		// The correct perception is 6.50; 7.00 is declared on purpose and the total to pay follows it.
		req.Summary.IVAPerception = 7.00
		req.Summary.TotalToPay = r2(d(req.Summary.TotalToPay).Add(d(0.50)))
		req.Summary.PaymentTypes = payments("03", req.Summary.TotalToPay, documentOptions{})
		req.Summary.TotalInWords = str(amountInWords(req.Summary.TotalToPay))
		client.probeEmit(t, "MSG-PERCEPTION", "/dte/ccf", req)
	})
}
