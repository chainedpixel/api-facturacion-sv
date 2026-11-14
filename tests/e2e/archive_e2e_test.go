//go:build e2e

package e2e

import (
	"testing"
)

// Archivo DTE (Normativa v2.0, regla 11 y Anexo I regla 4.1): cada documento emitido debe poder consultarse
// como Archivo DTE (estructura firmada + firmaElectronica + selloRecibido) y cada invalidacion debe quedar
// registrada como evento con su sello, su respuesta de Hacienda y su propio Archivo DTE.

func TestE2E_ArchivoDTE(t *testing.T) {
	client := newAPIClient(t)

	documents := []struct {
		name string
		path string
		body func() interface{}
	}{
		{"FE", "/dte/invoices", func() interface{} {
			return buildInvoice(invoiceSpec{receiver: companyReceiver(), items: []itemSpec{shoe("Zapato casual", 1, 45.00, 0)},
				opts: documentOptions{appendixes: nil}})
		}},
		{"FE consumidor final", "/dte/invoices", func() interface{} {
			return buildInvoice(invoiceSpec{items: []itemSpec{bread("Pan dulce", 4, 0.50, 0)}})
		}},
		{"CCF", "/dte/ccf", func() interface{} {
			return buildCCF(ccfSpec{receiver: ccfReceiver(), items: []itemSpec{bread("Pastel", 2, 20.00, 0)}})
		}},
		{"NR", "/dte/remissionnote", func() interface{} { return remissionFor(remissionReceiver()) }},
		{"FSE", "/dte/fse", func() interface{} {
			return buildFSE(fseSpec{receiver: excludedSubject(),
				items: []itemSpec{{description: "Servicio", itemType: 2, quantity: 1, unitPrice: 30.00}}})
		}},
		{"CR", "/dte/retention", func() interface{} {
			return buildRetention(companyReceiver(), []retentionLine{{documentNumber: "S221009001", documentType: "03",
				taxedAmount: 200.00, code: "22", description: "Compra"}}, todayDate())
		}},
	}

	var baseCCF *emittedDTE
	for _, doc := range documents {
		t.Run("archivo de "+doc.name, func(t *testing.T) {
			emitted := client.emit(t, doc.path, doc.body())
			if emitted == nil {
				t.Fatal("el documento no obtuvo sello")
			}
			client.archive(t, emitted)
			if doc.name == "CCF" {
				baseCCF = emitted
			}
		})
	}

	t.Run("archivo de NC", func(t *testing.T) {
		if baseCCF == nil {
			t.Skip("el CCF base no obtuvo sello")
		}
		note := client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: baseCCF,
			items: []itemSpec{bread("Devolucion", 1, 20.00, 0)}}))
		if note == nil {
			t.Fatal("la NC no obtuvo sello")
		}
		client.archive(t, note)
	})

	t.Run("invalidacion registrada como evento con su Archivo DTE", func(t *testing.T) {
		target := client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta a invalidar", 1, 33.00, 0)}}))
		if target == nil {
			t.Fatal("no se pudo emitir la FE")
		}
		if !client.invalidate(t, invalidation{target: target, kind: 2, reason: "Rescision para verificar el evento"}) {
			return
		}

		events := client.events(t, target.GenerationCode)
		if len(events) != 1 {
			t.Fatalf("se esperaba 1 evento registrado y hay %d", len(events))
		}
		event := events[0]
		if event.EventType != "INVALIDATION" || event.ReceptionStamp == "" || event.ProcessingDate == "" {
			t.Errorf("evento incompleto: tipo %q, sello %q, fecha %q", event.EventType, event.ReceptionStamp, event.ProcessingDate)
		}
		if len(event.HaciendaResponse) == 0 {
			t.Error("el evento no guardo la respuesta completa de Hacienda")
		}
		assertArchive(t, event.Archive, event.GenerationCode, event.ReceptionStamp)
		t.Logf("EVENTO %s | %s | sello %s | fhProcesamiento %s", event.EventType, event.GenerationCode, event.ReceptionStamp, event.ProcessingDate)
	})
}
