package notifier

import (
	"strings"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/notifier/email"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func setupRendererConfig(t *testing.T) {
	t.Helper()
	test.TestMain(t)
	config.Server.APIVersion = "3.0.0"
	config.Server.AmbientCode = "00"
}

func TestRenderContingencyContainsAllExpectedFields(t *testing.T) {
	setupRendererConfig(t)
	r, err := email.NewTemplateRenderer()
	if err != nil {
		t.Fatalf("renderer: %v", err)
	}

	evt := event.ContingencyActivatedEvent{
		BranchID:        7,
		ContingencyType: "1",
		Reason:          "MH no responde",
		AffectedDocs:    3,
		OccurredAtTime:  time.Date(2026, 5, 8, 14, 32, 5, 0, time.UTC),
	}
	subject, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(subject, "sucursal #7") {
		t.Errorf("subject missing branch: %q", subject)
	}
	for _, want := range []string{"API v3.0.0", "MH no responde", "Contingencia activada", "Ordo"} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(html, "<svg") {
		t.Error("html should contain inline SVG")
	}
	for _, want := range []string{"API v3.0.0", "MH no responde", "Sucursal: #7"} {
		if !strings.Contains(plain, want) {
			t.Errorf("plain missing %q", want)
		}
	}
}

func TestRenderEmissionFailureContainsDetails(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.EmissionFailureEvent{
		BranchID:       2,
		NIT:            "06140101001234",
		DTEType:        "01",
		ControlNumber:  "DTE-01-0001-000000001",
		GenerationCode: "ABCD-1234",
		ErrorCode:      "MH-403",
		LastError:      "Documento ya recibido",
		Attempts:       3,
		OccurredAtTime: time.Now(),
	}
	subject, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subject, "DTE-01-0001-000000001") || !strings.Contains(subject, "3 intento") {
		t.Errorf("subject should use ControlNumber and attempts, got %q", subject)
	}
	for _, want := range []string{"06140101001234", "MH-403", "Documento ya recibido", "API v3.0.0", "DTE-01-0001-000000001"} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(plain, "API v3.0.0") {
		t.Error("plain missing API v3.0.0")
	}
}

func TestRenderRetransmissionJobFailedShowsRejections(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.RetransmissionJobFailedEvent{
		JobName: "contingency_retransmission",
		FailedDocuments: []event.RejectedDocSummary{
			{
				ControlNumber: "DTE-01-0001-000000001",
				ErrorCode:     "MH-REJECT-01",
				Description:   "DTE inválido",
				Observations:  []string{"Campo emisor.nit incorrecto"},
			},
			{
				ControlNumber: "DTE-03-0001-000000002",
				ErrorCode:     "MH-REJECT-02",
				Description:   "DTE duplicado",
			},
		},
		OccurredAtTime: time.Now(),
	}
	subject, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subject, "2 documento") {
		t.Errorf("subject should mention document count, got %q", subject)
	}
	for _, want := range []string{
		"contingency_retransmission",
		"DTE-01-0001-000000001",
		"DTE-03-0001-000000002",
		"MH-REJECT-01",
		"DTE inválido",
		"Campo emisor.nit incorrecto",
		"API v3.0.0",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
	for _, want := range []string{"DTE-01-0001-000000001", "DTE inválido", "Campo emisor.nit incorrecto"} {
		if !strings.Contains(plain, want) {
			t.Errorf("plain missing %q", want)
		}
	}
}

func TestRenderRetransmissionJobInfraFailureShowsLastError(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.RetransmissionJobFailedEvent{
		JobName:        "contingency_retransmission",
		LastError:      "timeout reaching MH",
		OccurredAtTime: time.Now(),
	}
	subject, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subject, "fallo de ejecución") {
		t.Errorf("subject for infra failure should indicate execution error, got %q", subject)
	}
	for _, want := range []string{"contingency_retransmission", "timeout reaching MH", "API v3.0.0"} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(plain, "timeout reaching MH") {
		t.Error("plain missing last error")
	}
}

func TestRenderUsesConfiguredAPIVersion(t *testing.T) {
	setupRendererConfig(t)
	config.Server.APIVersion = "9.9.9"
	r, _ := email.NewTemplateRenderer()
	_, html, _, err := r.Render(event.ContingencyActivatedEvent{BranchID: 1, OccurredAtTime: time.Now()})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "API v9.9.9") {
		t.Error("html should reflect overridden API version")
	}
	config.Server.APIVersion = "3.0.0"
}

func TestContingencyTypeResolvesToLabel(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.ContingencyActivatedEvent{
		BranchID:        1,
		ContingencyType: "1",
		OccurredAtTime:  time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	label := "Servicio del Ministerio de Hacienda no disponible"
	if !strings.Contains(html, label) {
		t.Errorf("html should show contingency label, got raw %q — html snippet: %q",
			"1", extractSnippet(html, "contingencia"))
	}
	if !strings.Contains(plain, label) {
		t.Errorf("plain should show contingency label, got raw %q — plain snippet: %q",
			"1", extractSnippet(plain, "contingencia"))
	}
	if strings.Contains(html, ">1<") {
		t.Error("html must not show raw code '1' as the contingency type cell value")
	}
}

func TestAllContingencyTypesResolveToLabels(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	cases := []struct {
		typeCode string
		wantFrag string
	}{
		{"1", "Ministerio de Hacienda"},
		{"2", "sistemas internos"},
		{"3", "servicio de internet"},
		{"4", "servicio eléctrico"},
		{"5", "proceso de emisión"},
	}

	for _, tc := range cases {
		evt := event.ContingencyActivatedEvent{
			BranchID: 1, ContingencyType: tc.typeCode, OccurredAtTime: time.Now(),
		}
		_, html, plain, err := r.Render(evt)
		if err != nil {
			t.Fatalf("type %s: render error: %v", tc.typeCode, err)
		}
		if !strings.Contains(html, tc.wantFrag) {
			t.Errorf("type %s: html missing %q", tc.typeCode, tc.wantFrag)
		}
		if !strings.Contains(plain, tc.wantFrag) {
			t.Errorf("type %s: plain missing %q", tc.typeCode, tc.wantFrag)
		}
	}
}

func TestUnknownContingencyTypeFallsBackToRawValue(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.ContingencyActivatedEvent{
		BranchID: 1, ContingencyType: "99", OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render must not fail for unknown type: %v", err)
	}
	if !strings.Contains(html, "99") {
		t.Error("html should fall back to raw value '99' when label is unknown")
	}
	if !strings.Contains(plain, "99") {
		t.Error("plain should fall back to raw value '99' when label is unknown")
	}
}

func TestEmptyContingencyTypeDoesNotCrash(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.ContingencyActivatedEvent{BranchID: 1, ContingencyType: "", OccurredAtTime: time.Now()}
	_, _, _, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render must not crash with empty ContingencyType: %v", err)
	}
}

func TestDTETypeResolvesToLabel(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.EmissionFailureEvent{
		BranchID: 1, NIT: "00000000000000", DTEType: "01",
		GenerationCode: "X", Attempts: 1, OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(html, "Factura Electrónica") {
		t.Errorf("html should show DTE label; got snippet: %q", extractSnippet(html, "DTE"))
	}
	if !strings.Contains(html, "(01)") {
		t.Error("html should include raw code in parentheses: (01)")
	}
	if !strings.Contains(plain, "Factura Electrónica") {
		t.Error("plain should show DTE label")
	}
	if !strings.Contains(plain, "(01)") {
		t.Error("plain should include raw code in parentheses: (01)")
	}
}

func TestAllKnownDTETypesResolveToLabels(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	cases := []struct {
		code     string
		wantFrag string
	}{
		{"01", "Factura Electrónica"},
		{"03", "Crédito Fiscal"},
		{"04", "Nota de Remisión"},
		{"05", "Nota de Crédito"},
		{"06", "Nota de Débito"},
		{"07", "Retención"},
		{"08", "Liquidación"},
		{"09", "Contable de Liquidación"},
		{"11", "Exportación"},
		{"14", "Sujeto Excluido"},
		{"15", "Donación"},
	}

	for _, tc := range cases {
		evt := event.EmissionFailureEvent{
			BranchID: 1, NIT: "00000000000000", DTEType: tc.code,
			GenerationCode: "X", Attempts: 1, OccurredAtTime: time.Now(),
		}
		_, html, _, err := r.Render(evt)
		if err != nil {
			t.Fatalf("type %s: render error: %v", tc.code, err)
		}
		if !strings.Contains(html, tc.wantFrag) {
			t.Errorf("type %s: html missing label fragment %q", tc.code, tc.wantFrag)
		}
	}
}

func TestUnknownDTETypeFallsBackToRawCode(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.EmissionFailureEvent{
		BranchID: 1, NIT: "00000000000000", DTEType: "99",
		GenerationCode: "X", Attempts: 1, OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render must not fail for unknown DTE type: %v", err)
	}
	if !strings.Contains(html, "99") {
		t.Error("html should fall back to raw code '99'")
	}
	if !strings.Contains(plain, "99") {
		t.Error("plain should fall back to raw code '99'")
	}
}

func TestEmptyDTETypeDoesNotCrash(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.EmissionFailureEvent{
		BranchID: 1, NIT: "00000000000000", DTEType: "",
		GenerationCode: "X", Attempts: 1, OccurredAtTime: time.Now(),
	}
	_, _, _, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render must not crash with empty DTEType: %v", err)
	}
}

func TestContingencyNITAppearsInOutput(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.ContingencyActivatedEvent{
		BranchID: 5, NIT: "06140101991234", ContingencyType: "2", OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "06140101991234") {
		t.Error("html should include NIT when set")
	}
	if !strings.Contains(plain, "06140101991234") {
		t.Error("plain should include NIT when set")
	}
}

func TestContingencyClientNameAppearsInOutput(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.ContingencyActivatedEvent{
		BranchID: 1, ClientName: "Comercial El Salvador S.A.", ContingencyType: "3", OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Comercial El Salvador S.A.") {
		t.Error("html should include ClientName when set")
	}
	if !strings.Contains(plain, "Comercial El Salvador S.A.") {
		t.Error("plain should include ClientName when set")
	}
}

func TestContingencyBranchAddressAppearsInOutput(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.ContingencyActivatedEvent{
		BranchID: 3, BranchAddress: "San Salvador, San Salvador – Col. Escalón #45",
		ContingencyType: "4", OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Col. Escalón #45") {
		t.Error("html should include BranchAddress when set")
	}
	if !strings.Contains(plain, "Col. Escalón #45") {
		t.Error("plain should include BranchAddress when set")
	}
}

func TestContingencyEmptyOptionalFieldsProduceNoGarbage(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.ContingencyActivatedEvent{
		BranchID: 1, ContingencyType: "1", Reason: "test", OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(html, "&mdash;") {
		t.Error("html must not contain orphan &mdash; when BranchAddress is empty")
	}
	if strings.Contains(plain, " — \n") || strings.Contains(plain, "#1 —\n") {
		t.Error("plain must not contain orphan dash when BranchAddress is empty")
	}
	if strings.Contains(html, ">Cliente<") {
		t.Error("html must not render empty Cliente row")
	}
	if strings.Contains(html, "NIT del emisor") {
		t.Error("html must not render NIT row when NIT is empty")
	}
}

func TestEmissionClientNameAppearsInOutput(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.EmissionFailureEvent{
		BranchID: 1, NIT: "00000000000000", ClientName: "Distribuidora Norte S.A.",
		DTEType: "03", GenerationCode: "GEN-001", Attempts: 2, OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Distribuidora Norte S.A.") {
		t.Error("html should include ClientName when set")
	}
	if !strings.Contains(plain, "Distribuidora Norte S.A.") {
		t.Error("plain should include ClientName when set")
	}
}

func TestEmissionBranchAddressAppearsInOutput(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.EmissionFailureEvent{
		BranchID: 2, NIT: "00000000000000", BranchAddress: "Santa Ana, Santa Ana – Av. Independencia #8",
		DTEType: "01", GenerationCode: "GEN-002", Attempts: 1, OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Av. Independencia #8") {
		t.Error("html should include BranchAddress when set")
	}
	if !strings.Contains(plain, "Av. Independencia #8") {
		t.Error("plain should include BranchAddress when set")
	}
}

func TestEmissionEmptyOptionalFieldsProduceNoGarbage(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evt := event.EmissionFailureEvent{
		BranchID: 1, NIT: "00000000000000", DTEType: "01",
		GenerationCode: "X", Attempts: 1, OccurredAtTime: time.Now(),
	}
	_, html, plain, err := r.Render(evt)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(html, "&mdash;") {
		t.Error("html must not contain orphan &mdash; when BranchAddress is empty")
	}
	if strings.Contains(plain, " — \n") || strings.Contains(plain, "#1 —\n") {
		t.Error("plain must not contain orphan dash when BranchAddress is empty")
	}
	if strings.Contains(html, ">Cliente<") {
		t.Error("html must not render empty Cliente row")
	}
}

func TestAllTemplatesRenderWithMinimalEvent(t *testing.T) {
	setupRendererConfig(t)
	r, _ := email.NewTemplateRenderer()

	evts := []event.Event{
		event.ContingencyActivatedEvent{OccurredAtTime: time.Now()},
		event.EmissionFailureEvent{OccurredAtTime: time.Now()},
		event.RetransmissionJobFailedEvent{OccurredAtTime: time.Now()},
	}
	for _, evt := range evts {
		_, html, plain, err := r.Render(evt)
		if err != nil {
			t.Errorf("%s: render error with zero-value event: %v", evt.Name(), err)
			continue
		}
		if html == "" {
			t.Errorf("%s: html output is empty", evt.Name())
		}
		if plain == "" {
			t.Errorf("%s: plain output is empty", evt.Name())
		}
	}
}

func TestContingencyEventPayloadIncludesNewFields(t *testing.T) {
	evt := event.ContingencyActivatedEvent{
		BranchID:      3,
		NIT:           "06140101001111",
		ClientName:    "Test Corp",
		BranchAddress: "San Salvador – Calle X",
	}
	p := evt.Payload()
	checks := map[string]string{
		"nit":            "06140101001111",
		"client_name":    "Test Corp",
		"branch_address": "San Salvador – Calle X",
	}
	for key, want := range checks {
		got, ok := p[key]
		if !ok {
			t.Errorf("Payload() missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("Payload()[%q] = %v, want %v", key, got, want)
		}
	}
}

func TestEmissionEventPayloadIncludesNewFields(t *testing.T) {
	evt := event.EmissionFailureEvent{
		NIT:           "06140101002222",
		ClientName:    "Corp SA",
		BranchAddress: "Santa Ana – Calle Y",
	}
	p := evt.Payload()
	checks := map[string]string{
		"client_name":    "Corp SA",
		"branch_address": "Santa Ana – Calle Y",
	}
	for key, want := range checks {
		got, ok := p[key]
		if !ok {
			t.Errorf("Payload() missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("Payload()[%q] = %v, want %v", key, got, want)
		}
	}
}

func extractSnippet(s, keyword string) string {
	idx := strings.Index(strings.ToLower(s), strings.ToLower(keyword))
	if idx < 0 {
		if len(s) > 200 {
			return s[:200] + "…"
		}
		return s
	}
	start := idx - 60
	if start < 0 {
		start = 0
	}
	end := idx + 140
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}
