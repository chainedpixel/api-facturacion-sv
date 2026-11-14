//go:build e2e

package e2e

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// Invalidaciones reales (CAT-024: 1 error en la informacion, 2 rescision, 3 otro). Cada caso emite
// primero el documento a invalidar y, cuando aplica, el documento de reemplazo.

func TestE2E_Invalidacion(t *testing.T) {
	client := newAPIClient(t)

	simpleInvoice := func(t *testing.T, desc string, price float64) *emittedDTE {
		t.Helper()
		doc := client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe(desc, 1, price, 0)}}))
		if doc == nil {
			t.Fatal("no se pudo emitir el documento a invalidar")
		}
		return doc
	}
	simpleCCF := func(t *testing.T, desc string, price float64) *emittedDTE {
		t.Helper()
		doc := client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(), items: []itemSpec{shoe(desc, 2, price, 0)}}))
		if doc == nil {
			t.Fatal("no se pudo emitir el CCF a invalidar")
		}
		return doc
	}

	t.Run("FE tipo 2: rescision sin reemplazo", func(t *testing.T) {
		target := simpleInvoice(t, "Venta rescindida", 30.00)
		client.invalidate(t, invalidation{target: target, kind: 2, reason: "Rescision de la operacion"})
	})

	t.Run("FE tipo 1: error en la informacion con reemplazo", func(t *testing.T) {
		target := simpleInvoice(t, "Venta con precio errado", 31.00)
		replacement := simpleInvoice(t, "Venta corregida", 29.00)
		client.invalidate(t, invalidation{target: target, kind: 1, replacement: replacement,
			reason: "Precio ingresado incorrectamente"})
	})

	t.Run("FE tipo 3: otro motivo con reemplazo", func(t *testing.T) {
		target := simpleInvoice(t, "Venta con cambio de producto", 45.00)
		replacement := simpleInvoice(t, "Venta del producto cambiado", 45.00)
		client.invalidate(t, invalidation{target: target, kind: 3, replacement: replacement,
			reason: "Cambio de producto solicitado por el cliente"})
	})

	t.Run("CCF tipo 1 con reemplazo", func(t *testing.T) {
		target := simpleCCF(t, "Venta con NRC errado", 40.00)
		replacement := simpleCCF(t, "Venta corregida", 40.00)
		client.invalidate(t, invalidation{target: target, kind: 1, replacement: replacement,
			reason: "Datos del receptor incorrectos"})
	})

	t.Run("NC y luego su CCF (orden correcto de dependencias)", func(t *testing.T) {
		ccf := simpleCCF(t, "Venta que se anulara completa", 50.00)
		note := client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{shoe("Devolucion parcial", 1, 50.00, 0)}}))
		if note == nil {
			t.Fatal("no se pudo emitir la NC")
		}
		if client.invalidate(t, invalidation{target: note, kind: 2, reason: "Se rescinde la nota de credito"}) {
			client.invalidate(t, invalidation{target: ccf, kind: 2, reason: "Se rescinde la operacion completa"})
		}
	})

	t.Run("CCF con NC vigente debe rechazarse (invalidar primero la nota)", func(t *testing.T) {
		// Normativa 13.1 y diagnostico seccion 7: un CCFE con NCE/NDE aplicadas exige invalidar primero las notas
		ccf := simpleCCF(t, "Venta con nota vigente", 60.00)
		note := client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{shoe("Devolucion", 1, 60.00, 0)}}))
		if note == nil {
			t.Fatal("no se pudo emitir la NC")
		}
		client.expectInvalidationRejected(t, invalidation{target: ccf, kind: 2, reason: "Intento con nota vigente"},
			"PendingAdjustmentNotes", note.GenerationCode)
	})

	t.Run("FSE tipo 2", func(t *testing.T) {
		target := client.emit(t, "/dte/fse", buildFSE(fseSpec{receiver: excludedSubject(),
			items: []itemSpec{{description: "Servicio a anular", itemType: 2, quantity: 1, unitPrice: 25.00}}}))
		if target == nil {
			t.Fatal("no se pudo emitir la FSE")
		}
		client.invalidate(t, invalidation{target: target, kind: 2, reason: "Servicio no prestado"})
	})

	t.Run("NR tipo 2", func(t *testing.T) {
		target := client.emit(t, "/dte/remissionnote", remissionFor(remissionReceiver()))
		if target == nil {
			t.Fatal("no se pudo emitir la NR")
		}
		client.invalidate(t, invalidation{target: target, kind: 2, reason: "Traslado cancelado"})
	})

	t.Run("documento ya invalidado no puede invalidarse de nuevo", func(t *testing.T) {
		target := simpleInvoice(t, "Venta a invalidar dos veces", 22.00)
		if client.invalidate(t, invalidation{target: target, kind: 2, reason: "Primera invalidacion"}) {
			client.expectInvalidationRejected(t, invalidation{target: target, kind: 2, reason: "Segunda invalidacion"})
		}
	})
}

// TestE2E_InvalidacionNotaCredito verifies that credit notes are invalidated without a replacement for types 1 and 3
// and that the API rejects a replacement code for them before transmitting to Hacienda.
func TestE2E_InvalidacionNotaCredito(t *testing.T) {
	client := newAPIClient(t)

	emitNote := func(t *testing.T, price float64) *emittedDTE {
		t.Helper()
		ccf := client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Venta base de la nota", 1, price, 0)}}))
		if ccf == nil {
			t.Fatal("no se pudo emitir el CCF base")
		}
		note := client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{shoe("Devolucion parcial", 1, price, 0)}}))
		if note == nil {
			t.Fatal("no se pudo emitir la NC")
		}
		return note
	}

	t.Run("NC tipo 1 sin reemplazo", func(t *testing.T) {
		note := emitNote(t, 35.00)
		client.invalidate(t, invalidation{target: note, kind: 1, reason: "Error en la informacion de la nota"})
	})

	t.Run("NC tipo 3 sin reemplazo", func(t *testing.T) {
		note := emitNote(t, 36.00)
		client.invalidate(t, invalidation{target: note, kind: 3, reason: "Otro motivo para anular la nota"})
	})

	t.Run("NC tipo 1 con reemplazo se rechaza antes de transmitir", func(t *testing.T) {
		note := emitNote(t, 37.00)
		replacement := emitNote(t, 38.00)
		client.expectInvalidationRejected(t, invalidation{target: note, kind: 1, replacement: replacement,
			reason: "Intento con reemplazo"}, "debe ser nulo")
	})
}

// TestE2E_InvalidacionNotaDebitoSaldo verifies that invalidating a debit note subtracts its amount from the CCF
// balance: after the ND of 20.00 is invalidated, the available balance returns to the original 100.00.
func TestE2E_InvalidacionNotaDebitoSaldo(t *testing.T) {
	client := newAPIClient(t)

	ccf := client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
		items: []itemSpec{shoe("Venta base con saldo", 1, 100.00, 0)}}))
	if ccf == nil {
		t.Fatal("no se pudo emitir el CCF base")
	}
	debit := client.emit(t, "/dte/debitnote", buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
		items: []itemSpec{shoe("Cargo adicional", 1, 20.00, 0)}}))
	if debit == nil {
		t.Fatal("no se pudo emitir la ND")
	}
	if !client.invalidate(t, invalidation{target: debit, kind: 2, reason: "Se rescinde el cargo adicional"}) {
		t.Fatal("no se pudo invalidar la ND")
	}

	creditNote := func(price float64) *structs.CreateCreditNoteRequest {
		return buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{shoe("Devolucion contra el saldo", 1, price, 0)}})
	}

	t.Run("NC de 110.00 excede el saldo de 100.00", func(t *testing.T) {
		client.expectRejectedAny(t, "/dte/creditnote", creditNote(110.00), "InvalidCreditNoteTransaction", "remaining balance", "saldo")
	})

	t.Run("NC de 100.00 dentro del saldo", func(t *testing.T) {
		client.emit(t, "/dte/creditnote", creditNote(100.00))
	})
}
