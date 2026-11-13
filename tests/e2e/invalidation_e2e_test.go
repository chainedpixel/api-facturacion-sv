//go:build e2e

package e2e

import (
	"testing"
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
		client.expectInvalidationRejected(t, invalidation{target: ccf, kind: 2, reason: "Intento con nota vigente"})
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
