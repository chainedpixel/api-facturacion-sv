//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Deadline probe in two phases, to be run on different days:
//
//  1. E2E_DEADLINE_PROBE_PHASE=seal emits a CCF and a remission note and stores their data in the evidence file.
//  2. E2E_DEADLINE_PROBE_PHASE=invalidate reads the file and invalidates both with type 2, logging whether
//     Hacienda accepted or rejected each event. Hacienda rejections are reported, not failed.
//
// The file lives in E2E_EVIDENCE_DIR, or in tests/e2e/testdata when it is empty.

type probeDocument struct {
	GenerationCode string `json:"generation_code"`
	ControlNumber  string `json:"control_number"`
	EmissionDate   string `json:"emission_date"`
	ReceptionStamp string `json:"reception_stamp"`
}

type probeFile struct {
	SealedOn string        `json:"sealed_on"`
	CCF      probeDocument `json:"ccf"`
	NR       probeDocument `json:"nr"`
}

func probeFilePath() string {
	dir := os.Getenv("E2E_EVIDENCE_DIR")
	if dir == "" {
		dir = "testdata"
	}
	return filepath.Join(dir, "deadline_probe.json")
}

func toProbeDocument(doc *emittedDTE) probeDocument {
	return probeDocument{GenerationCode: doc.GenerationCode, ControlNumber: doc.ControlNumber,
		EmissionDate: doc.EmissionDate, ReceptionStamp: doc.ReceptionStamp}
}

func (p probeDocument) toEmitted() *emittedDTE {
	return &emittedDTE{GenerationCode: p.GenerationCode, ControlNumber: p.ControlNumber,
		EmissionDate: p.EmissionDate, ReceptionStamp: p.ReceptionStamp}
}

// TestE2E_SondeoPlazosInvalidacion probes the CCF and NR invalidation deadlines on later dates.
func TestE2E_SondeoPlazosInvalidacion(t *testing.T) {
	switch os.Getenv("E2E_DEADLINE_PROBE_PHASE") {
	case "seal":
		deadlineProbeSeal(t)
	case "invalidate":
		deadlineProbeInvalidate(t)
	default:
		t.Skip("define E2E_DEADLINE_PROBE_PHASE=seal o invalidate (ver comentario del archivo)")
	}
}

func deadlineProbeSeal(t *testing.T) {
	client := newAPIClient(t)

	ccf := client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
		items: []itemSpec{shoe("CCF sonda de plazos", 1, 40.00, 0)}}))
	nr := client.emit(t, "/dte/remissionnote", remissionFor(remissionReceiver()))
	if ccf == nil || nr == nil {
		t.Fatal("no se pudieron emitir los documentos de la sonda")
	}

	content, err := json.MarshalIndent(probeFile{SealedOn: time.Now().Format(time.RFC3339),
		CCF: toProbeDocument(ccf), NR: toProbeDocument(nr)}, "", "  ")
	if err != nil {
		t.Fatalf("no se pudo serializar la sonda: %v", err)
	}
	path := probeFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("no se pudo crear el directorio de la sonda: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("no se pudo guardar la sonda: %v", err)
	}
	t.Logf("SONDA guardada en %s", path)
}

func deadlineProbeInvalidate(t *testing.T) {
	client := newAPIClient(t)

	content, err := os.ReadFile(probeFilePath())
	if err != nil {
		t.Fatalf("no hay sonda de la fase seal: %v", err)
	}
	var probe probeFile
	if err := json.Unmarshal(content, &probe); err != nil {
		t.Fatalf("sonda ilegible: %v", err)
	}
	t.Logf("sonda sellada el %s", probe.SealedOn)

	targets := []struct {
		name string
		doc  probeDocument
	}{{"CCF", probe.CCF}, {"NR", probe.NR}}
	for _, target := range targets {
		t.Run(target.name+" tipo 2 fuera de la fecha de sellado", func(t *testing.T) {
			status, resp, raw := client.post(t, "/dte/invalidation",
				invalidationRequest(invalidation{target: target.doc.toEmitted(), kind: 2, reason: "Sonda de plazos de invalidacion"}))
			if status == http.StatusOK || status == http.StatusCreated {
				t.Logf("SONDA %s ACEPTADA | %s | sellado %s | respuesta: %s", target.name, target.doc.GenerationCode, probe.SealedOn, raw)
				return
			}
			if status >= 500 && resp.Error == nil {
				t.Errorf("error de transporte o servidor (HTTP %d): %s", status, raw)
				return
			}
			t.Logf("SONDA %s RECHAZADA (HTTP %d) | %s | sellado %s | %s | respuesta: %s", target.name, status,
				target.doc.GenerationCode, probe.SealedOn, describeError(resp, raw), raw)
		})
	}
}
