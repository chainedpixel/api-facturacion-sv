//go:build e2e

package e2e

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// TestE2E_NotasPagosMixtosEInvalidacionNC covers the strict item rule of the remission note, the strict item rule of
// credit notes, the debit note balance that only caps the non-subject sale, mixed payments with condition 3 and the
// credit note correction order: invalidate the wrong note with a null replacement and then emit the right one.
func TestE2E_NotasPagosMixtosEInvalidacionNC(t *testing.T) {
	client := newAPIClient(t)

	ccf := client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
		items: []itemSpec{shoe("Venta base para notas", 1, 50.00, 0), nonSubject("Venta no sujeta base", 1, 20.00)}}))
	if ccf == nil {
		t.Fatal("no se pudo emitir el CCF base")
	}

	t.Run("NC con un item sin documento relacionado se rechaza localmente", func(t *testing.T) {
		req := buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{shoe("Devolucion con item suelto", 1, 10.00, 0), shoe("Segundo item de la devolucion", 1, 5.00, 0)}})
		req.Items[1].RelatedDoc = nil
		client.expectRejectedAny(t, "/dte/creditnote", req, "MissingItemRelatedDoc", "RelatedDoc")
	})

	remissionRelatedTo := func(desc string) *structs.CreateRemissionNoteRequest {
		req := buildRemissionNote(&structs.RemissionNoteReceiverRequest{ReceiverRequest: remissionReceiver(), BienTitulo: str("04")},
			[]itemSpec{shoe(desc, 2, 25.00, 0), shoe("Segundo articulo remitido", 1, 10.00, 0)})
		related := relatedDoc(ccf, "03")
		req.RelatedDocs = []*structs.RelatedDocRequest{&related[0]}
		return req
	}

	t.Run("NR que relaciona el CCF con todos los items vinculados obtiene sello", func(t *testing.T) {
		req := remissionRelatedTo("Mercaderia facturada en el CCF")
		for i := range req.Items {
			req.Items[i].DocumentNumber = str(ccf.GenerationCode)
		}
		client.emit(t, "/dte/remissionnote", req)
	})

	t.Run("NR que relaciona el CCF con un item vinculado y otro en null se rechaza localmente", func(t *testing.T) {
		req := remissionRelatedTo("Mercaderia con item suelto")
		req.Items[0].DocumentNumber = str(ccf.GenerationCode)
		client.expectRejected(t, "/dte/remissionnote", req, "MissingItemRelatedDoc")
	})

	t.Run("NR que relaciona el CCF sin que ningun item lo referencie se rechaza localmente", func(t *testing.T) {
		client.expectRejected(t, "/dte/remissionnote", remissionRelatedTo("Mercaderia sin vinculo"), "MissingItemRelatedDoc")
	})

	t.Run("NC previa que consume el saldo gravado del CCF", func(t *testing.T) {
		client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{shoe("Devolucion de casi todo lo gravado", 1, 45.00, 0)}}))
	})

	t.Run("ND gravada mayor que el saldo restante del CCF obtiene sello", func(t *testing.T) {
		client.emit(t, "/dte/debitnote", buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{shoe("Cargo gravado mayor al saldo", 1, 120.00, 0)}}))
	})

	t.Run("ND no sujeta mayor que el total no sujeto del CCF se rechaza localmente", func(t *testing.T) {
		client.expectRejected(t, "/dte/debitnote", buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: ccf,
			items: []itemSpec{nonSubject("Cargo no sujeto excedido", 1, 25.00)}}), "InvalidDebitNoteNotSubjectAmount")
	})

	t.Run("FE con condicion 3, un pago 01 y otro 13 que suman el total obtiene sello", func(t *testing.T) {
		req := buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta mixta con saldo por cobrar", 2, 30.00, 0)},
			opts:  documentOptions{paymentCodes: []string{"01", "13"}}})
		req.Summary.OperationCondition = 3
		client.emit(t, "/dte/invoices", req)
	})

	t.Run("NC tipo 1 se invalida con reemplazo null y despues se emite la NC correcta", func(t *testing.T) {
		base := client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Venta base para corregir la NC", 1, 60.00, 0)}}))
		if base == nil {
			t.Fatal("no se pudo emitir el CCF base")
		}
		wrong := client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
			items: []itemSpec{shoe("Devolucion con monto incorrecto", 1, 40.00, 0)}}))
		if wrong == nil {
			t.Fatal("no se pudo emitir la NC incorrecta")
		}
		if !client.invalidate(t, invalidation{target: wrong, kind: 1, reason: "Monto de la devolucion incorrecto"}) {
			t.Fatal("no se pudo invalidar la NC incorrecta")
		}
		client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
			items: []itemSpec{shoe("Devolucion con monto correcto", 1, 55.00, 0)}}))
	})
}
