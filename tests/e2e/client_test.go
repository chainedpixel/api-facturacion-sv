//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// Cliente de la suite E2E. Se habla con la API real del microservicio (que debe estar corriendo),
// la cual firma y transmite cada documento al ambiente de pruebas de Hacienda (MH_AMBIENT_CODE=00).
// Configuracion en .env: E2E_API_URL, E2E_API_KEY, E2E_API_SECRET, E2E_MH_USER, E2E_MH_PASSWORD.

type apiClient struct {
	baseURL     string
	token       string
	http        *http.Client
	evidenceDir string
}

type apiError struct {
	Message string   `json:"message"`
	Details []string `json:"details"`
	Code    string   `json:"code"`
}

type apiResponse struct {
	Success        bool            `json:"success"`
	ReceptionStamp *string         `json:"reception_stamp"`
	Data           json.RawMessage `json:"data"`
	Error          *apiError       `json:"error"`
}

// emittedDTE contiene los datos de un documento emitido que otros casos necesitan (NC, ND, invalidacion)
type emittedDTE struct {
	GenerationCode string
	ControlNumber  string
	EmissionDate   string
	ReceptionStamp string
}

// loadDotEnv lee KEY=VALUE del .env de la raiz sin sobrescribir variables ya definidas en el entorno
func loadDotEnv(t *testing.T) {
	t.Helper()
	file, err := os.Open(filepath.Join(utils.FindProjectRoot(), ".env"))
	if err != nil {
		t.Fatalf("no se pudo abrir .env: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, strings.TrimSpace(value))
		}
	}
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("falta la variable %s en .env", key)
	}
	return value
}

// newAPIClient inicia sesion con las credenciales de prueba y devuelve un cliente autenticado
func newAPIClient(t *testing.T) *apiClient {
	t.Helper()
	loadDotEnv(t)

	client := &apiClient{
		baseURL: strings.TrimRight(requireEnv(t, "E2E_API_URL"), "/"),
		// La API reintenta la transmision a Hacienda; se deja margen para esos reintentos
		http:        &http.Client{Timeout: 90 * time.Second},
		evidenceDir: os.Getenv("E2E_EVIDENCE_DIR"),
	}
	if client.evidenceDir != "" {
		if err := os.MkdirAll(client.evidenceDir, 0o700); err != nil {
			t.Fatalf("no se pudo crear E2E_EVIDENCE_DIR: %v", err)
		}
	}

	login := map[string]interface{}{
		"api_key":    requireEnv(t, "E2E_API_KEY"),
		"api_secret": requireEnv(t, "E2E_API_SECRET"),
		"credentials": map[string]string{
			"username": requireEnv(t, "E2E_MH_USER"),
			"password": requireEnv(t, "E2E_MH_PASSWORD"),
		},
	}

	status, resp, raw := client.post(t, "/auth/login", login)
	if status != http.StatusOK || !resp.Success {
		t.Fatalf("login fallido (HTTP %d): %s", status, raw)
	}
	if err := json.Unmarshal(resp.Data, &client.token); err != nil || client.token == "" {
		t.Fatalf("login sin token: %s", raw)
	}

	return client
}

func (c *apiClient) post(t *testing.T, path string, body interface{}) (int, apiResponse, []byte) {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("no se pudo serializar el request: %v", err)
	}

	status, raw := c.do(t, http.MethodPost, path, payload)
	var resp apiResponse
	_ = json.Unmarshal(raw, &resp)
	return status, resp, raw
}

func (c *apiClient) do(t *testing.T, method, path string, payload []byte) (int, []byte) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		t.Fatalf("request invalido: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("la API no respondio (%s): %v", path, err)
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	c.recordEvidence(t, method, path, payload, res.StatusCode, raw)
	return res.StatusCode, raw
}

type evidenceRecord struct {
	Test       string          `json:"test"`
	RecordedAt string          `json:"recorded_at"`
	Method     string          `json:"method"`
	Path       string          `json:"path"`
	Request    json.RawMessage `json:"request"`
	HTTPStatus int             `json:"http_status"`
	Response   json.RawMessage `json:"response"`
}

func (c *apiClient) recordEvidence(t *testing.T, method, path string, payload []byte, status int, raw []byte) {
	if c.evidenceDir == "" {
		return
	}
	request := json.RawMessage("null")
	response := json.RawMessage(raw)
	if strings.HasPrefix(path, "/auth/") {
		request = json.RawMessage(`"credentials omitted"`)
		response = json.RawMessage(`"token omitted"`)
	} else {
		if json.Valid(payload) {
			request = payload
		}
		if !json.Valid(raw) {
			quoted, _ := json.Marshal(string(raw))
			response = quoted
		}
	}

	record, err := json.MarshalIndent(evidenceRecord{
		Test: t.Name(), RecordedAt: time.Now().Format(time.RFC3339), Method: method, Path: path,
		Request: request, HTTPStatus: status, Response: response,
	}, "", "  ")
	if err != nil {
		t.Logf("no se pudo serializar la evidencia: %v", err)
		return
	}
	name := fmt.Sprintf("%s_%s.json", time.Now().Format("20060102T150405.000000"), evidenceFileName(t.Name()))
	if err := os.WriteFile(filepath.Join(c.evidenceDir, name), record, 0o600); err != nil {
		t.Logf("no se pudo guardar la evidencia: %v", err)
	}
}

func evidenceFileName(testName string) string {
	var b strings.Builder
	for _, r := range testName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	name := b.String()
	if len(name) > 120 {
		name = name[:120]
	}
	return name
}

// emit envia un documento y exige que Hacienda devuelva sello de recepcion
func (c *apiClient) emit(t *testing.T, path string, body interface{}) *emittedDTE {
	t.Helper()

	status, resp, raw := c.post(t, path, body)
	if status != http.StatusCreated || resp.ReceptionStamp == nil || *resp.ReceptionStamp == "" {
		t.Errorf("Hacienda no otorgo sello (HTTP %d): %s", status, describeError(resp, raw))
		return nil
	}

	emitted := &emittedDTE{ReceptionStamp: *resp.ReceptionStamp}
	var document struct {
		Identificacion struct {
			CodigoGeneracion string `json:"codigoGeneracion"`
			NumeroControl    string `json:"numeroControl"`
			FecEmi           string `json:"fecEmi"`
		} `json:"identificacion"`
	}
	if err := json.Unmarshal(resp.Data, &document); err == nil {
		emitted.GenerationCode = document.Identificacion.CodigoGeneracion
		emitted.ControlNumber = document.Identificacion.NumeroControl
		emitted.EmissionDate = document.Identificacion.FecEmi
	}
	// La respuesta de creacion es el Archivo DTE (estructura + firmaElectronica + selloRecibido)
	assertArchive(t, resp.Data, emitted.GenerationCode, emitted.ReceptionStamp)

	t.Logf("SELLO %s | %s | %s", emitted.ReceptionStamp, emitted.ControlNumber, emitted.GenerationCode)
	return emitted
}

// expectRejected exige que la API rechace el documento (HTTP 4xx, sin sello) y que la respuesta
// contenga el codigo de error esperado; wantCode vacio acepta cualquier rechazo
func (c *apiClient) expectRejected(t *testing.T, path string, body interface{}, wantCode string) {
	t.Helper()
	if wantCode == "" {
		c.expectRejectedAny(t, path, body)
		return
	}
	c.expectRejectedAny(t, path, body, wantCode)
}

// expectRejectedAny exige el rechazo de la API (HTTP 4xx, sin sello) y que la respuesta contenga, sin distinguir
// mayusculas, al menos uno de los fragmentos indicados (codigo o texto del mensaje); sin fragmentos acepta cualquier rechazo
func (c *apiClient) expectRejectedAny(t *testing.T, path string, body interface{}, wantAny ...string) {
	t.Helper()

	status, resp, raw := c.post(t, path, body)
	if resp.ReceptionStamp != nil && *resp.ReceptionStamp != "" {
		t.Errorf("se esperaba rechazo pero Hacienda otorgo sello %s", *resp.ReceptionStamp)
		return
	}
	if status < 400 || status >= 500 {
		t.Errorf("se esperaba HTTP 4xx y se obtuvo %d: %s", status, raw)
		return
	}
	if len(wantAny) > 0 && !containsAny(string(raw), wantAny) {
		t.Errorf("rechazo sin ninguno de los fragmentos esperados %q: %s", wantAny, describeError(resp, raw))
		return
	}
	t.Logf("RECHAZO esperado (HTTP %d): %s", status, describeError(resp, raw))
}

func containsAny(text string, fragments []string) bool {
	lowered := strings.ToLower(text)
	for _, fragment := range fragments {
		if strings.Contains(lowered, strings.ToLower(fragment)) {
			return true
		}
	}
	return false
}

// invalidation describe un evento de invalidacion (CAT-024: 1 error, 2 rescision, 3 otro)
type invalidation struct {
	target      *emittedDTE
	kind        int
	replacement *emittedDTE
	reason      string
}

// invalidationRequest arma el cuerpo de POST /dte/invalidation con el reemplazo cuando se indica
func invalidationRequest(inv invalidation) map[string]interface{} {
	req := map[string]interface{}{
		"generation_code": inv.target.GenerationCode,
		"reason": map[string]interface{}{
			"type": inv.kind, "reason_field": inv.reason,
			"responsible_name": "Responsable de Pruebas", "responsible_doc_type": "36",
			"responsible_num_doc": os.Getenv("E2E_MH_USER"),
			"requestor_name":      os.Getenv("E2E_RECEPTOR_NAME"), "requestor_doc_type": "36",
			"requestor_num_doc": os.Getenv("E2E_RECEPTOR_NIT"),
		},
	}
	if inv.replacement != nil {
		req["replacement_generation_code"] = inv.replacement.GenerationCode
	}
	return req
}

// invalidate transmite el evento; devuelve true si la API y Hacienda lo aceptaron
func (c *apiClient) invalidate(t *testing.T, inv invalidation) bool {
	t.Helper()

	req := invalidationRequest(inv)

	status, resp, raw := c.post(t, "/dte/invalidation", req)
	if status >= 300 || !resp.Success {
		t.Errorf("invalidacion rechazada (HTTP %d): %s", status, describeError(resp, raw))
		return false
	}

	// Normativa v2.0, regla 11: el evento se entrega al receptor como Archivo DTE con su sello
	var event eventResponse
	if err := json.Unmarshal(resp.Data, &event); err != nil {
		t.Errorf("respuesta de invalidacion ilegible: %v", err)
		return false
	}
	if event.ReceptionStamp == "" {
		t.Errorf("la invalidacion no devolvio el sello de Hacienda: %s", resp.Data)
	}
	assertArchive(t, event.Archive, event.GenerationCode, event.ReceptionStamp)
	t.Logf("INVALIDATION aceptada | tipo %d | %s | sello %s", inv.kind, inv.target.GenerationCode, event.ReceptionStamp)
	return true
}

// eventResponse es la respuesta de un evento (invalidacion) o de GET /dte/{id}/eventos
type eventResponse struct {
	GenerationCode   string          `json:"generation_code"`
	EventType        string          `json:"event_type"`
	RelatedDTEID     *string         `json:"related_dte_id"`
	ReceptionStamp   string          `json:"reception_stamp"`
	ProcessingDate   string          `json:"processing_date"`
	HaciendaResponse json.RawMessage `json:"hacienda_response"`
	Archive          json.RawMessage `json:"archivo_dte"`
}

// getJSON hace un GET autenticado y devuelve el codigo HTTP y el cuerpo
func (c *apiClient) getJSON(t *testing.T, path string) (int, []byte) {
	t.Helper()
	return c.do(t, http.MethodGet, path, nil)
}

// events consulta GET /dte/{id}/eventos
func (c *apiClient) events(t *testing.T, generationCode string) []eventResponse {
	t.Helper()
	status, raw := c.getJSON(t, "/dte/"+generationCode+"/eventos")
	if status != http.StatusOK {
		t.Errorf("GET eventos devolvio HTTP %d: %s", status, raw)
		return nil
	}
	var resp struct {
		Data []eventResponse `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Errorf("respuesta de eventos ilegible: %v", err)
	}
	return resp.Data
}

// archive consulta GET /dte/{id}/archivo y verifica el Archivo DTE del documento
func (c *apiClient) archive(t *testing.T, doc *emittedDTE) {
	t.Helper()
	status, raw := c.getJSON(t, "/dte/"+doc.GenerationCode+"/archivo")
	if status != http.StatusOK {
		t.Errorf("GET archivo devolvio HTTP %d: %s", status, raw)
		return
	}
	assertArchive(t, raw, doc.GenerationCode, doc.ReceptionStamp)
}

// assertArchive verifica el Archivo DTE (Anexo I, regla 4.1): estructura firmada integra (igual al payload
// del JWS), firmaElectronica, selloRecibido, y que el sello ya no se inserte en el apendice
func assertArchive(t *testing.T, archive json.RawMessage, generationCode, stamp string) {
	t.Helper()
	if len(archive) == 0 {
		t.Errorf("no se recibio Archivo DTE para %s", generationCode)
		return
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(archive, &doc); err != nil {
		t.Errorf("Archivo DTE no es JSON valido: %v", err)
		return
	}

	var signature, sello string
	_ = json.Unmarshal(doc["firmaElectronica"], &signature)
	_ = json.Unmarshal(doc["selloRecibido"], &sello)
	if signature == "" || strings.Count(signature, ".") != 2 {
		t.Errorf("firmaElectronica ausente o invalida en el Archivo DTE de %s", generationCode)
		return
	}
	if stamp != "" && sello != stamp {
		t.Errorf("selloRecibido %q no coincide con el sello %q", sello, stamp)
	}

	var identification struct {
		CodigoGeneracion string `json:"codigoGeneracion"`
	}
	_ = json.Unmarshal(doc["identificacion"], &identification)
	if identification.CodigoGeneracion != generationCode {
		t.Errorf("el Archivo DTE trae codigoGeneracion %q y se esperaba %q", identification.CodigoGeneracion, generationCode)
	}

	// La estructura del archivo debe ser exactamente el payload firmado: archivo - firma - sello == payload
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(strings.Split(signature, ".")[1], "="))
	if err != nil {
		t.Errorf("payload del JWS ilegible: %v", err)
		return
	}
	delete(doc, "firmaElectronica")
	delete(doc, "selloRecibido")
	withoutEvidence, _ := json.Marshal(doc)
	var a, b interface{}
	_ = json.Unmarshal(withoutEvidence, &a)
	_ = json.Unmarshal(payload, &b)
	if fmt.Sprint(a) != fmt.Sprint(b) {
		t.Errorf("la estructura del Archivo DTE de %s no coincide con lo firmado", generationCode)
	}

	if strings.Contains(string(doc["apendice"]), "Sello de recep") {
		t.Errorf("el sello no debe insertarse en el apendice (Archivo DTE de %s)", generationCode)
	}
}

// expectInvalidationRejected expects the invalidation to be rejected with an error containing every wantTexts fragment.
func (c *apiClient) expectInvalidationRejected(t *testing.T, inv invalidation, wantTexts ...string) {
	t.Helper()

	req := invalidationRequest(inv)
	status, resp, raw := c.post(t, "/dte/invalidation", req)
	if status < 300 && resp.Success {
		t.Errorf("se esperaba rechazo de la invalidacion de %s y fue aceptada", inv.target.GenerationCode)
		return
	}
	for _, text := range wantTexts {
		if !strings.Contains(strings.ToLower(string(raw)), strings.ToLower(text)) {
			t.Errorf("el rechazo no contiene %q: %s", text, describeError(resp, raw))
		}
	}
	t.Logf("RECHAZO esperado de invalidacion (HTTP %d): %s", status, describeError(resp, raw))
}

// emitInContingency envia un documento cuando la API simula indisponibilidad de Hacienda: se espera
// que lo acepte (HTTP 2xx) sin sello y que quede pendiente para el evento de contingencia
func (c *apiClient) emitInContingency(t *testing.T, path string, body interface{}) *emittedDTE {
	t.Helper()

	status, resp, raw := c.post(t, path, body)
	if status >= 300 {
		t.Errorf("la API no acepto el documento en contingencia (HTTP %d): %s", status, describeError(resp, raw))
		return nil
	}
	if resp.ReceptionStamp != nil && *resp.ReceptionStamp != "" {
		t.Errorf("se esperaba contingencia y Hacienda otorgo sello: revise FORCE_CONTINGENCY")
		return nil
	}
	emitted := &emittedDTE{}
	var document struct {
		Identificacion struct {
			CodigoGeneracion string `json:"codigoGeneracion"`
			NumeroControl    string `json:"numeroControl"`
			FecEmi           string `json:"fecEmi"`
			TipoModelo       int    `json:"tipoModelo"`
			TipoOperacion    int    `json:"tipoOperacion"`
		} `json:"identificacion"`
	}
	if err := json.Unmarshal(resp.Data, &document); err == nil {
		emitted.GenerationCode = document.Identificacion.CodigoGeneracion
		emitted.ControlNumber = document.Identificacion.NumeroControl
		emitted.EmissionDate = document.Identificacion.FecEmi
		t.Logf("CONTINGENCIA | %s | tipoModelo %d tipoOperacion %d", emitted.GenerationCode,
			document.Identificacion.TipoModelo, document.Identificacion.TipoOperacion)
	}
	return emitted
}

// consultStatus consulta un DTE en la API y devuelve su sello (vacio si aun no lo tiene) y su estado
func (c *apiClient) consultStatus(t *testing.T, generationCode string) (stamp, status string) {
	t.Helper()

	_, raw := c.do(t, http.MethodGet, "/dte/"+generationCode, nil)

	var resp struct {
		Data struct {
			ReceptionStamp *string `json:"reception_stamp"`
			Status         string  `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &resp)
	if resp.Data.ReceptionStamp != nil {
		stamp = *resp.Data.ReceptionStamp
	}
	return stamp, resp.Data.Status
}

func describeError(resp apiResponse, raw []byte) string {
	if resp.Error == nil {
		return string(raw)
	}
	return fmt.Sprintf("[%s] %s | %s", resp.Error.Code, resp.Error.Message, strings.Join(resp.Error.Details, " | "))
}
