//go:build e2e

package e2e

import (
	"os"
	"testing"
)

// TestE2E_PlazosDocumentosAntiguos checks deadline rules with documents whose real stamp is months old.
// E2E_OLD_CCF is a CCF stamped before the previous month, E2E_OLD_CCF_RECEIVER_NIT its receiver NIT and
// E2E_OLD_FE an FE stamped more than three months ago. The test is skipped when they are not set.
func TestE2E_PlazosDocumentosAntiguos(t *testing.T) {
	oldCCF := os.Getenv("E2E_OLD_CCF")
	oldFE := os.Getenv("E2E_OLD_FE")
	if oldCCF == "" && oldFE == "" {
		t.Skip("define E2E_OLD_CCF y/o E2E_OLD_FE con documentos sellados hace meses")
	}
	client := newAPIClient(t)

	if oldCCF != "" {
		t.Run("CCF despues del decimo dia habil del mes siguiente", func(t *testing.T) {
			client.expectInvalidationRejected(t, invalidation{target: &emittedDTE{GenerationCode: oldCCF}, kind: 2,
				reason: "Intento fuera de plazo"}, "InvalidationDeadlineExpired")
		})
		t.Run("NC sobre un CCF de mas de 3 meses", func(t *testing.T) {
			receiver := adjustmentReceiver()
			if nit := os.Getenv("E2E_OLD_CCF_RECEIVER_NIT"); nit != "" {
				receiver.NIT, receiver.DocumentNumber = str(nit), str(nit)
			}
			client.expectRejected(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: receiver,
				related: &emittedDTE{GenerationCode: oldCCF, EmissionDate: todayDate()},
				items:   []itemSpec{shoe("Devolucion fuera de plazo", 1, 10.00, 0)}}), "RelatedDocOutOfPeriod")
		})
	}

	if oldFE != "" {
		t.Run("FE pasados 3 meses del sello", func(t *testing.T) {
			client.expectInvalidationRejected(t, invalidation{target: &emittedDTE{GenerationCode: oldFE}, kind: 2,
				reason: "Intento fuera de plazo"}, "InvalidationDeadlineExpired")
		})
	}
}
