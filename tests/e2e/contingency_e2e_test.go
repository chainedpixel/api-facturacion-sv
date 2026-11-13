//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Contingencia en dos fases (requiere reiniciar la API entre fases):
//
//  1. API con FORCE_CONTINGENCY=true (simula Hacienda caida en ambiente 00):
//     E2E_CONTINGENCY_PHASE=1 go test -tags e2e ./tests/e2e/ -run TestE2E_Contingencia -v
//     Emite documentos que quedan pendientes y guarda sus codigos de generacion.
//  2. API con FORCE_CONTINGENCY=false: el job (cada 5 minutos en pruebas) transmite el evento de
//     contingencia y retransmite los documentos.
//     E2E_CONTINGENCY_PHASE=2 go test -tags e2e ./tests/e2e/ -run TestE2E_Contingencia -v -timeout 20m
//     Espera a que cada documento obtenga sello.

func contingencyFile() string {
	return filepath.Join(os.TempDir(), "e2e_contingency_codes.txt")
}

func TestE2E_Contingencia(t *testing.T) {
	switch os.Getenv("E2E_CONTINGENCY_PHASE") {
	case "1":
		contingencyPhaseOne(t)
	case "2":
		contingencyPhaseTwo(t)
	default:
		t.Skip("contingencia se ejecuta por fases con E2E_CONTINGENCY_PHASE=1 o 2 (ver comentario del archivo)")
	}
}

func contingencyPhaseOne(t *testing.T) {
	client := newAPIClient(t)

	var codes []string
	record := func(doc *emittedDTE) {
		if doc != nil && doc.GenerationCode != "" {
			codes = append(codes, doc.GenerationCode)
		}
	}

	t.Run("FE en contingencia", func(t *testing.T) {
		record(client.emitInContingency(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta durante caida de Hacienda", 2, 25.00, 0)}})))
	})
	t.Run("FE consumidor final en contingencia", func(t *testing.T) {
		record(client.emitInContingency(t, "/dte/invoices", buildInvoice(invoiceSpec{
			items: []itemSpec{bread("Pan dulce", 10, 0.40, 0)}})))
	})
	t.Run("CCF en contingencia", func(t *testing.T) {
		record(client.emitInContingency(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{bread("Pedido de pan", 30, 1.10, 0)}})))
	})
	t.Run("FSE en contingencia", func(t *testing.T) {
		record(client.emitInContingency(t, "/dte/fse", buildFSE(fseSpec{receiver: excludedSubject(),
			items: []itemSpec{{description: "Servicio durante contingencia", itemType: 2, quantity: 1, unitPrice: 30.00}}})))
	})

	if len(codes) == 0 {
		t.Fatal("ningun documento quedo en contingencia")
	}
	if err := os.WriteFile(contingencyFile(), []byte(strings.Join(codes, "\n")), 0o600); err != nil {
		t.Fatalf("no se pudieron guardar los codigos: %v", err)
	}
	t.Logf("%d documentos en contingencia guardados en %s", len(codes), contingencyFile())
}

func contingencyPhaseTwo(t *testing.T) {
	client := newAPIClient(t)

	raw, err := os.ReadFile(contingencyFile())
	if err != nil {
		t.Fatalf("no hay documentos de la fase 1: %v", err)
	}
	pending := map[string]bool{}
	for _, code := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if code != "" {
			pending[code] = true
		}
	}

	// Plazo amplio: el job corre cada 5 minutos y primero debe obtener sello el evento de contingencia
	deadline := time.Now().Add(16 * time.Minute)
	for len(pending) > 0 && time.Now().Before(deadline) {
		for code := range pending {
			stamp, status := client.consultStatus(t, code)
			if stamp != "" {
				t.Logf("SELLO tras contingencia %s | %s | estado %s", stamp, code, status)
				// El JWS del lote se guarda al firmar: el documento debe tener su Archivo DTE
				client.archive(t, &emittedDTE{GenerationCode: code, ReceptionStamp: stamp})
				delete(pending, code)
			}
		}
		if len(pending) > 0 {
			time.Sleep(30 * time.Second)
		}
	}

	for code := range pending {
		_, status := client.consultStatus(t, code)
		t.Errorf("el documento %s no obtuvo sello tras la contingencia (estado %s)", code, status)
	}
}
