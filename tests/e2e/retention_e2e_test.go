//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// Comprobante de Retencion (07) con documentos fisicos y notas sobre un CR. El receptor del CR es el
// sujeto de retencion (el proveedor), por eso se usa el contribuyente receptor de pruebas.

func todayDate() string { return time.Now().Format("2006-01-02") }

func TestE2E_ComprobanteRetencion(t *testing.T) {
	client := newAPIClient(t)
	today := time.Now().Format("2006-01-02")

	var retention *emittedDTE
	t.Run("CR con dos documentos fisicos al 1%", func(t *testing.T) {
		retention = client.emit(t, "/dte/retention", buildRetention(companyReceiver(), []retentionLine{
			{documentNumber: "S221001347", documentType: "03", taxedAmount: 500.00, code: "22",
				description: "Compra de materia prima"},
			{documentNumber: "S221001348", documentType: "03", taxedAmount: 120.00, code: "22",
				description: "Servicio de mantenimiento"},
		}, today))
	})

	t.Run("CR al 13% (codigo C4: IVA 13% = monto sujeto x 13%)", func(t *testing.T) {
		client.emit(t, "/dte/retention", buildRetention(companyReceiver(), []retentionLine{
			{documentNumber: "S221001350", documentType: "03", taxedAmount: 300.00, code: "C4",
				description: "Compra a sujeto con retencion total"},
		}, today))
	})

	t.Run("CR con codigos de retencion distintos debe rechazarse", func(t *testing.T) {
		// Hacienda: "codigoRetencionMH NO SE PERMITEN VALORES DIFERENTES"
		client.expectRejected(t, "/dte/retention", buildRetention(companyReceiver(), []retentionLine{
			{documentNumber: "S221001351", documentType: "03", taxedAmount: 500.00, code: "22", description: "Compra"},
			{documentNumber: "S221001352", documentType: "03", taxedAmount: 120.00, code: "C4", description: "Servicio"},
		}, today), "MixedRetentionCodes")
	})

	t.Run("CR con un solo documento fisico al 1%", func(t *testing.T) {
		client.emit(t, "/dte/retention", buildRetention(companyReceiver(), []retentionLine{
			{documentNumber: "S221001349", documentType: "01", taxedAmount: 1250.75, code: "22",
				description: "Compra de cuero"},
		}, today))
	})

	// Exploratorio: la normativa (campos 100 y 145) permite ajustar un CR con NC/ND donde las ventas
	// van en 0.00 y se ajusta el IVA 13% o el IVA retenido. El modelo actual de NC calcula el IVA desde la
	// venta gravada, por lo que se registra el resultado sin fallar la suite.
	t.Run("exploratorio: NC sobre un CR", func(t *testing.T) {
		if retention == nil {
			t.Skip("el CR no obtuvo sello")
		}
		status, resp, raw := client.post(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{
			receiver: adjustmentReceiver(), related: retention, relatedType: "07", ivaRetention: true,
			items: []itemSpec{shoe("Ajuste de retencion", 1, 100.00, 0)},
		}))
		if resp.ReceptionStamp != nil {
			t.Logf("EXPLORATORIO NC sobre CR: aceptada con sello %s", *resp.ReceptionStamp)
			return
		}
		t.Logf("EXPLORATORIO NC sobre CR: no soportada (HTTP %d): %s", status, describeError(resp, raw))
	})
}
