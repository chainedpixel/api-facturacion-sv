//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	record := func(t *testing.T, doc *emittedDTE) {
		if doc != nil && doc.GenerationCode != "" {
			codes = append(codes, fmt.Sprintf("%s\t%d\t%s", doc.GenerationCode, doc.ContingencyType, doc.SignedDocument))
			assertPendingContingencyDocument(t, client, doc)
		}
	}

	t.Run("FE en contingencia", func(t *testing.T) {
		record(t, client.emitInContingency(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta durante caida de Hacienda", 2, 25.00, 0)}})))
	})
	t.Run("FE consumidor final en contingencia", func(t *testing.T) {
		record(t, client.emitInContingency(t, "/dte/invoices", buildInvoice(invoiceSpec{
			items: []itemSpec{bread("Pan dulce", 10, 0.40, 0)}})))
	})
	t.Run("CCF en contingencia", func(t *testing.T) {
		record(t, client.emitInContingency(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{bread("Pedido de pan", 30, 1.10, 0)}})))
	})
	t.Run("NR en contingencia", func(t *testing.T) {
		record(t, client.emitInContingency(t, "/dte/remissionnote", remissionFor(remissionReceiver())))
	})
	t.Run("FSE en contingencia", func(t *testing.T) {
		record(t, client.emitInContingency(t, "/dte/fse", buildFSE(fseSpec{receiver: excludedSubject(),
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
	type delivered struct {
		contingencyType int
		signedDocument  string
	}
	pending := map[string]delivered{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 || fields[0] == "" {
			continue
		}
		contingencyType, _ := strconv.Atoi(fields[1])
		pending[fields[0]] = delivered{contingencyType: contingencyType, signedDocument: fields[2]}
	}

	// Plazo amplio: el job corre cada 5 minutos y primero debe obtener sello el evento de contingencia
	deadline := time.Now().Add(16 * time.Minute)
	for len(pending) > 0 && time.Now().Before(deadline) {
		for code, sent := range pending {
			stamp, status := client.consultStatus(t, code)
			if status == "EXPIRED" {
				t.Errorf("el documento %s vencio (EXPIRED): no se transmitio dentro de las 72 horas del evento", code)
				delete(pending, code)
				continue
			}
			if stamp != "" {
				t.Logf("SELLO tras contingencia %s | %s | estado %s", stamp, code, status)
				// El JWS del lote se guarda al firmar: el documento debe tener su Archivo DTE
				client.archive(t, &emittedDTE{GenerationCode: code, ReceptionStamp: stamp})
				assertDeferredIdentification(t, client, code, stamp, sent.contingencyType, sent.signedDocument)
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

// assertDeferredIdentification checks that the stamped document of a contingency emission is the one delivered to the
// consumer (same signature) and carries the deferred model (2), the contingency transmission type (2) and the
// contingency type informed when it was generated.
func assertDeferredIdentification(t *testing.T, client *apiClient, code, stamp string, contingencyType int, signedDocument string) {
	t.Helper()
	status, raw := client.getJSON(t, "/dte/"+code+"/archivo")
	if status != http.StatusOK {
		t.Errorf("GET archivo de %s devolvio HTTP %d", code, status)
		return
	}
	var archive struct {
		FirmaElectronica string `json:"firmaElectronica"`
		SelloRecibido    string `json:"selloRecibido"`
		Identificacion   struct {
			TipoModelo       int     `json:"tipoModelo"`
			TipoOperacion    int     `json:"tipoOperacion"`
			TipoContingencia *int    `json:"tipoContingencia"`
			MotivoContin     *string `json:"motivoContin"`
		} `json:"identificacion"`
	}
	if err := json.Unmarshal(raw, &archive); err != nil {
		t.Errorf("Archivo DTE de %s no es JSON valido: %v", code, err)
		return
	}
	ident := archive.Identificacion
	if ident.TipoModelo != 2 || ident.TipoOperacion != 2 || ident.TipoContingencia == nil {
		t.Errorf("el documento sellado %s no se transmitio como diferido: tipoModelo=%d tipoOperacion=%d tipoContingencia=%v",
			code, ident.TipoModelo, ident.TipoOperacion, ident.TipoContingencia)
		return
	}
	if *ident.TipoContingencia != contingencyType {
		t.Errorf("el documento sellado %s trae tipoContingencia %d y se entrego con %d", code, *ident.TipoContingencia, contingencyType)
	}
	if *ident.TipoContingencia == 5 && (ident.MotivoContin == nil || *ident.MotivoContin == "") {
		t.Errorf("el documento sellado %s con tipo de contingencia 5 no trae motivoContin", code)
	}
	if archive.SelloRecibido != stamp {
		t.Errorf("el Archivo DTE de %s trae selloRecibido %q y el sello consultado es %q", code, archive.SelloRecibido, stamp)
	}
	if archive.FirmaElectronica != signedDocument {
		t.Errorf("la firma del documento sellado %s no es la que se entrego al consumidor", code)
	}
	t.Logf("DIFERIDO %s | tipoModelo=%d tipoOperacion=%d tipoContingencia=%d", code, ident.TipoModelo, ident.TipoOperacion, *ident.TipoContingencia)
}

// assertPendingContingencyDocument checks that a document emitted in contingency is pending without a reception
// stamp and that invalidating it is rejected locally because Hacienda has not received it yet.
func assertPendingContingencyDocument(t *testing.T, client *apiClient, doc *emittedDTE) {
	t.Helper()
	stamp, status := client.consultStatus(t, doc.GenerationCode)
	if status != "PENDING" {
		t.Errorf("el documento en contingencia %s debe estar PENDING y esta %q", doc.GenerationCode, status)
	}
	if stamp != "" {
		t.Errorf("el documento en contingencia %s no debe tener sello de recepcion y trae %q", doc.GenerationCode, stamp)
	}
	client.expectInvalidationRejected(t, invalidation{target: doc, kind: 2, reason: "Intento sobre documento en contingencia"}, "DocumentPending")
}
