package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/batch"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

func advObsJWS(payload string) string {
	return "eyJhbGciOiJSUzUxMiJ9." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".c2lnbmF0dXJl"
}

func advObsStamp(value string) *string { return &value }

func advObsJSONList(items []string) string {
	raw, _ := json.Marshal(items)
	return string(raw)
}

// TestAdversarialObservations_ParserHostileInputs checks the parser never panics, never returns nil and only returns
// strings that were present in the input.
func TestAdversarialObservations_ParserHostileInputs(t *testing.T) {
	huge := make([]string, 5000)
	for i := range huge {
		huge[i] = fmt.Sprintf("observation %d", i)
	}
	long := strings.Repeat("a", 10000)

	cases := []struct {
		name     string
		raw      string
		expected []string
	}{
		{"html and script", `{"observaciones":["<script>alert(1)</script>","<b>x</b>"]}`, []string{"<script>alert(1)</script>", "<b>x</b>"}},
		{"quotes and backslashes", `{"observaciones":["say \"hi\"","back\\slash","\\\\"]}`, []string{`say "hi"`, `back\slash`, `\\`}},
		{"newlines and tabs", `{"observaciones":["a\nb","c\td","e\r\nf"]}`, []string{"a\nb", "c\td", "e\r\nf"}},
		{"control characters", `{"observaciones":["\u0000\u0001\u001f","ok"]}`, []string{"\x00\x01\x1f", "ok"}},
		{"unicode escapes", `{"observaciones":["é日本","😀"]}`, []string{"é日本", "\U0001F600"}},
		{"lone surrogate", `{"observaciones":["\ud800","ok"]}`, []string{"�", "ok"}},
		{"10k characters", `{"observaciones":["` + long + `"]}`, []string{long}},
		{"5000 items", `{"observaciones":` + advObsJSONList(huge) + `}`, huge},
		{"non string items mixed in", `{"observaciones":["a",1,"b",null,{"x":"y"},["z"],true]}`, nil},
		{"only non string items", `{"observaciones":[1,2,3]}`, []string{}},
		{"null items", `{"observaciones":[null,"a",null]}`, []string{"a"}},
		{"bom prefix", string(rune(0xFEFF)) + `{"observaciones":["a"]}`, []string{}},
		{"duplicated keys last wins", `{"observaciones":["a"],"observaciones":["b","c"]}`, []string{"b", "c"}},
		{"duplicated keys last empty", `{"observaciones":["a"],"observaciones":[]}`, []string{}},
		{"duplicated keys last null", `{"observaciones":["a"],"observaciones":null}`, []string{}},
		{"case insensitive key", `{"OBSERVACIONES":["a"]}`, []string{"a"}},
		{"nested key is ignored", `{"data":{"observaciones":["nested"]}}`, []string{}},
		{"trailing garbage", `{"observaciones":["a"]} trailing`, []string{}},
		{"two documents", `{"observaciones":["a"]}{"observaciones":["b"]}`, []string{}},
		{"deeply nested", strings.Repeat(`{"a":`, 5000) + `1` + strings.Repeat(`}`, 5000), []string{}},
		{"numbers only", `12345`, []string{}},
		{"literal null", `null`, []string{}},
		{"literal true", `true`, []string{}},
		{"whitespace items kept", `{"observaciones":[" ","\t"]}`, []string{" ", "\t"}},
		{"nul byte raw", "{\x00}", []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var result []string
			require.NotPanics(t, func() { result = dte.ObservationsFromHaciendaResponse(tc.raw) })
			require.NotNil(t, result)

			encoded, err := json.Marshal(result)
			require.NoError(t, err)
			assert.NotEqual(t, "null", string(encoded))
			assert.True(t, json.Valid(encoded))
			for _, observation := range result {
				assert.NotEmpty(t, observation, "empty observations are dropped")
			}
			if tc.expected != nil {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestAdversarialObservations_ArchiveResponseNeverBreaksMarshalling checks every stored response shape serializes
// and the archive only appears with a non empty stamp and a valid JWS.
func TestAdversarialObservations_ArchiveResponseNeverBreaksMarshalling(t *testing.T) {
	test.TestMain(t)

	validJWS := advObsJWS(`{"identificacion":{"codigoGeneracion":"X"}}`)
	huge := strings.Repeat("z", 5*1024*1024)
	hugeJSON := `{"observaciones":["` + huge + `"]}`

	cases := []struct {
		name        string
		signed      string
		stamp       *string
		response    string
		wantArchive bool
		wantObs     []string
	}{
		{"valid jws and stamp", validJWS, advObsStamp("STAMP"), `{"observaciones":["a"]}`, true, []string{"a"}},
		{"stamp pointer to empty string", validJWS, advObsStamp(""), `{"observaciones":["a"]}`, false, []string{"a"}},
		{"nil stamp", validJWS, nil, `{"observaciones":["a"]}`, false, []string{"a"}},
		{"whitespace stamp is still a stamp", validJWS, advObsStamp(" "), ``, true, []string{}},
		{"signed doc not a jws", "not-a-jws", advObsStamp("STAMP"), ``, false, []string{}},
		{"signed doc two parts", "a.b", advObsStamp("STAMP"), ``, false, []string{}},
		{"signed doc four parts", "a.b.c.d", advObsStamp("STAMP"), ``, false, []string{}},
		{"signed doc payload not base64", "a.@@@.c", advObsStamp("STAMP"), ``, false, []string{}},
		{"signed doc payload not an object", advObsJWS(`[1,2]`), advObsStamp("STAMP"), ``, false, []string{}},
		{"signed doc empty payload", advObsJWS(``), advObsStamp("STAMP"), ``, false, []string{}},
		{"signed doc empty", "", advObsStamp("STAMP"), ``, false, []string{}},
		{"stored response json string", validJWS, advObsStamp("S"), `"just a string"`, true, []string{}},
		{"stored response json array", validJWS, advObsStamp("S"), `["a","b"]`, true, []string{}},
		{"stored response json number", validJWS, advObsStamp("S"), `12345`, true, []string{}},
		{"stored response json null", validJWS, advObsStamp("S"), `null`, true, []string{}},
		{"stored response json true", validJWS, advObsStamp("S"), `true`, true, []string{}},
		{"stored response plain text", validJWS, advObsStamp("S"), `<html>gateway error</html>`, true, []string{}},
		{"stored response truncated", validJWS, advObsStamp("S"), `{"observaciones":["a"`, true, []string{}},
		{"stored response with bom", validJWS, advObsStamp("S"), string(rune(0xFEFF)) + `{"observaciones":["a"]}`, true, []string{}},
		{"stored response with invalid utf8", validJWS, advObsStamp("S"), "bad \xff\xfe bytes", true, []string{}},
		{"stored response with control characters", validJWS, advObsStamp("S"), "bad \x00\x01 bytes", true, []string{}},
		{"stored response with html", validJWS, advObsStamp("S"), `{"observaciones":["<script>alert(1)</script>"]}`, true, []string{"<script>alert(1)</script>"}},
		{"stored response very large", validJWS, advObsStamp("S"), hugeJSON, true, []string{huge}},
		{"stored response very large invalid", validJWS, advObsStamp("S"), huge, true, []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := dte.DTEEvent{
				GenerationCode: "EVENT", EventType: dte.EventInvalidation, Status: constants.DocumentReceived,
				SignedDocument: tc.signed, ReceptionStamp: tc.stamp, HaciendaResponse: tc.response, CreatedAt: time.Now(),
			}

			var result dte.EventArchiveResponse
			require.NotPanics(t, func() { result = dte_documents.ToEventArchiveResponse(event) })
			assert.Equal(t, tc.wantObs, result.Observations)

			raw, err := json.Marshal(result)
			require.NoError(t, err, "serialization must never fail")
			require.True(t, json.Valid(raw))

			var decoded map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &decoded))
			require.Contains(t, decoded, "observations")
			assert.NotEqual(t, "null", string(decoded["observations"]))

			if tc.wantArchive {
				require.Contains(t, decoded, "archivo_dte")
				var archive map[string]interface{}
				require.NoError(t, json.Unmarshal(decoded["archivo_dte"], &archive))
				assert.Equal(t, tc.signed, archive["firmaElectronica"])
				assert.Equal(t, *tc.stamp, archive["selloRecibido"])
			} else {
				assert.NotContains(t, decoded, "archivo_dte")
			}

			if tc.response == "" {
				assert.NotContains(t, decoded, "hacienda_response")
			} else {
				require.Contains(t, decoded, "hacienda_response")
				if json.Valid([]byte(tc.response)) && tc.response != "null" {
					var expected, actual interface{}
					require.NoError(t, json.Unmarshal([]byte(tc.response), &expected))
					require.NoError(t, json.Unmarshal(decoded["hacienda_response"], &actual))
					assert.Equal(t, expected, actual, "valid stored JSON is kept as is")
				} else if !json.Valid([]byte(tc.response)) {
					var text string
					require.NoError(t, json.Unmarshal(decoded["hacienda_response"], &text), "invalid text is kept as a JSON string")
					if utf8Valid(tc.response) {
						assert.Equal(t, tc.response, text)
					}
				}
			}
		})
	}
}

func utf8Valid(value string) bool {
	return strings.ToValidUTF8(value, "") == value
}

// TestAdversarialObservations_ArchiveWithMalformedJWSPayloadIsOmitted checks a JWS whose payload only looks
// like an object (braces) does not make the event impossible to serialize.
func TestAdversarialObservations_ArchiveWithMalformedJWSPayloadIsOmitted(t *testing.T) {
	test.TestMain(t)

	payloads := []string{`{not json}`, `{"a":}`, `{"a":1,}`, `{'a':1}`, `{"a":"unterminated}`}
	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			event := dte.DTEEvent{
				GenerationCode: "EVENT", SignedDocument: advObsJWS(payload), ReceptionStamp: advObsStamp("STAMP"),
				HaciendaResponse: `{"observaciones":["a"]}`,
			}
			result := dte_documents.ToEventArchiveResponse(event)
			_, err := json.Marshal(result)
			assert.NoError(t, err, "payload %q yields an invalid archivo_dte", payload)
		})
	}
}

// TestAdversarialObservations_ArchiveHugeCountOfEventsKeepsOrder checks the per event mapping is independent.
func TestAdversarialObservations_ArchiveHugeCountOfEventsKeepsOrder(t *testing.T) {
	test.TestMain(t)

	jws := advObsJWS(`{"a":1}`)
	for i := 0; i < 2000; i++ {
		stamp := fmt.Sprintf("STAMP-%d", i)
		var stampPointer *string
		if i%2 == 0 {
			stampPointer = &stamp
		}
		result := dte_documents.ToEventArchiveResponse(dte.DTEEvent{
			GenerationCode: fmt.Sprintf("E-%d", i), SignedDocument: jws, ReceptionStamp: stampPointer,
			HaciendaResponse: fmt.Sprintf(`{"observaciones":["obs-%d"]}`, i),
		})
		assert.Equal(t, []string{fmt.Sprintf("obs-%d", i)}, result.Observations)
		assert.Equal(t, i%2 == 0, result.Archive != nil, "archive only for stamped events (%d)", i)
	}
}

func advObsEventBody(t *testing.T, harness *contingencyEventHarness, status int, body string) {
	t.Helper()
	serveContingency(t, status, body)
	require.NotPanics(t, func() {
		_ = harness.service.PrepareAndSendContingencyEvent(context.Background(), contingencyEvidenceDocs())
	})
}

// TestAdversarialObservations_ContingencyEventDefinitiveRejectionsStoreOneRejectedEvent checks that definitive
// rejections store exactly one REJECTED event, transient answers none, and duplicates none.
func TestAdversarialObservations_ContingencyEventDefinitiveRejectionsStoreOneRejectedEvent(t *testing.T) {
	test.TestMain(t)

	hugeObservations := make([]string, 5000)
	for i := range hugeObservations {
		hugeObservations[i] = fmt.Sprintf("obs %d", i)
	}
	cases := []struct {
		name   string
		status int
		body   string
		want   int
	}{
		{"definitive 200 rechazado", http.StatusOK, `{"estado":"RECHAZADO","mensaje":"x"}`, 1},
		{"definitive 200 rechazado without message", http.StatusOK, `{"estado":"RECHAZADO"}`, 1},
		{"definitive 200 rechazado with nested object", http.StatusOK, `{"estado":"RECHAZADO","data":{"estado":"PROCESADO","selloRecibido":"S"}}`, 1},
		{"definitive 200 rechazado with hostile observation", http.StatusOK, `{"estado":"RECHAZADO","observaciones":["<script>x</script>","\"q\"\n"]}`, 1},
		{"definitive 200 rechazado with 10k observation", http.StatusOK, `{"estado":"RECHAZADO","observaciones":["` + strings.Repeat("o", 10000) + `"]}`, 1},
		{"definitive 200 rechazado with 5000 observations", http.StatusOK, `{"estado":"RECHAZADO","observaciones":` + advObsJSONList(hugeObservations) + `}`, 1},
		{"definitive 200 rechazado duplicated keys", http.StatusOK, `{"estado":"PROCESADO","estado":"RECHAZADO"}`, 1},
		{"definitive 400 no superadas lower", http.StatusBadRequest, `{"mensaje":"validaciones no superadas"}`, 1},
		{"definitive 400 no superadas upper", http.StatusBadRequest, `{"mensaje":"VALIDACIONES NO SUPERADAS"}`, 1},
		{"definitive 400 no superadas mixed", http.StatusBadRequest, `{"mensaje":"Las Validaciones No Superadas"}`, 1},
		{"definitive 400 no superadas with estado rechazado", http.StatusBadRequest, `{"estado":"RECHAZADO","mensaje":"No Superadas"}`, 1},
		{"definitive 422 no superadas", http.StatusUnprocessableEntity, `{"mensaje":"no superadas","observaciones":["a"]}`, 1},
		{"transient 503", http.StatusServiceUnavailable, `{"mensaje":"temporarily unavailable"}`, 0},
		{"transient 500 without message", http.StatusInternalServerError, `{"estado":"ERROR"}`, 0},
		{"transient 500 empty object", http.StatusInternalServerError, `{}`, 0},
		{"transient 400 array body", http.StatusBadRequest, `["no superadas"]`, 0},
		{"transient 400 string body", http.StatusBadRequest, `"no superadas"`, 0},
		{"transient 400 number body", http.StatusBadRequest, `123`, 0},
		{"transient 400 null body", http.StatusBadRequest, `null`, 0},
		{"transient 400 truncated body", http.StatusBadRequest, `{"estado":"RECHAZADO"`, 0},
		{"transient 400 other message", http.StatusBadRequest, `{"estado":"PROCESADO","mensaje":"other"}`, 0},
		{"transient 400 observation mentions no superadas", http.StatusBadRequest, `{"mensaje":"other","observaciones":["no superadas"]}`, 0},
		{"transient 400 message is an array", http.StatusBadRequest, `{"mensaje":["no superadas"]}`, 0},
		{"transient 502 html", http.StatusBadGateway, `<html>no superadas</html>`, 0},
		{"transient empty body", http.StatusBadRequest, ``, 0},
		{"duplicate by message", http.StatusBadRequest, `{"mensaje":"Ya existe evento de contingencia"}`, 0},
		{"duplicate by typo message", http.StatusBadRequest, `{"mensaje":"ya existe envento"}`, 0},
		{"duplicate rechazado body by observation", http.StatusOK, `{"estado":"RECHAZADO","mensaje":"x","observaciones":["Ya existe evento"]}`, 0},
		{"duplicate wins over no superadas", http.StatusBadRequest, `{"estado":"RECHAZADO","mensaje":"ya existe evento, validaciones no superadas"}`, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			harness := newContingencyEventHarness(t, ctrl, &evidenceSigner{}, nil)
			harness.repo.EXPECT().MarkEventStamped(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			advObsEventBody(t, harness, tc.status, tc.body)

			require.Len(t, harness.received, tc.want)
			for _, event := range harness.received {
				assert.Equal(t, constants.DocumentRejected, event.Status)
				assert.Equal(t, dte.EventContingency, event.EventType)
				assert.Equal(t, evidenceSignedJWS, event.SignedDocument)
				assert.Nil(t, event.ReceptionStamp)
				assert.True(t, json.Valid([]byte(event.HaciendaResponse)))
			}
		})
	}
}

// TestAdversarialObservations_ContingencyEventAmbiguousRejectionsStoreAtMostOne checks the cases whose classification
// is debatable never store more than one event and never store them as received.
func TestAdversarialObservations_ContingencyEventAmbiguousRejectionsStoreAtMostOne(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"lowercase estado", http.StatusBadRequest, `{"estado":"rechazado","mensaje":"other"}`},
		{"estado with spaces", http.StatusBadRequest, `{"estado":" RECHAZADO ","mensaje":"other"}`},
		{"estado as number", http.StatusBadRequest, `{"estado":1,"mensaje":"no superadas"}`},
		{"estado as array", http.StatusBadRequest, `{"estado":["RECHAZADO"],"mensaje":"no superadas"}`},
		{"estado null", http.StatusBadRequest, `{"estado":null,"mensaje":"no superadas"}`},
		{"200 lowercase estado", http.StatusOK, `{"estado":"rechazado"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			harness := newContingencyEventHarness(t, ctrl, &evidenceSigner{}, nil)
			harness.repo.EXPECT().MarkEventStamped(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			advObsEventBody(t, harness, tc.status, tc.body)

			assert.LessOrEqual(t, len(harness.received), 1)
			if tc.status != http.StatusOK {
				for _, event := range harness.received {
					assert.Equal(t, constants.DocumentRejected, event.Status)
				}
			}
		})
	}
}

// TestAdversarialObservations_DefinitiveRejectionWithNonStringMessageIsStored checks that a rejection reported
// with a non text message is still kept as a rejected event.
func TestAdversarialObservations_DefinitiveRejectionWithNonStringMessageIsStored(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"message as object", http.StatusOK, `{"estado":"RECHAZADO","mensaje":{"detail":"x"}}`},
		{"message as array", http.StatusOK, `{"estado":"RECHAZADO","mensaje":["x"]}`},
		{"message as number", http.StatusOK, `{"estado":"RECHAZADO","mensaje":123}`},
		{"message as null with observations", http.StatusOK, `{"estado":"RECHAZADO","mensaje":null,"observaciones":["x"]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			harness := newContingencyEventHarness(t, ctrl, &evidenceSigner{}, nil)
			harness.repo.EXPECT().MarkEventStamped(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			advObsEventBody(t, harness, tc.status, tc.body)

			require.Len(t, harness.received, 1, "estado RECHAZADO is a definitive rejection whatever the message type")
			assert.Equal(t, constants.DocumentRejected, harness.received[0].Status)
		})
	}
}

type advObsBatchCall struct {
	ids          []string
	observations []string
	results      map[string]dte.ReceptionResult
	status       string
}

func advObsRunBatch(t *testing.T, responseBody string, docs map[string]dte.ContingencyDocument) []advObsBatchCall {
	t.Helper()
	test.TestMain(t)

	server := httptest.NewServer(&evidenceBatchHacienda{response: responseBody})
	defer server.Close()
	previous := config.MHPaths.LoteReceptionConsultURL
	config.MHPaths.LoteReceptionConsultURL = server.URL
	defer func() { config.MHPaths.LoteReceptionConsultURL = previous }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockContingencyRepositoryPort(ctrl)
	seq := mocks.NewMockSequentialNumberManager(ctrl)

	var calls []advObsBatchCall
	repo.EXPECT().UpdateBatch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "batch-1", "mh-1", gomock.Any()).
		DoAndReturn(func(_ context.Context, ids, observations []string, results map[string]dte.ReceptionResult, _, _, status string) error {
			calls = append(calls, advObsBatchCall{ids, observations, results, status})
			return nil
		}).AnyTimes()
	seq.EXPECT().ConfirmReservationByDocumentID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	seq.EXPECT().ReleaseReservationByDocumentID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	service := batch.NewBatchTransmitterService(&stubHaciendaAuth{}, nil, repo, seq, minimalTransmissionConfig(),
		&noopTimeProvider{}, evidenceDBConnection(t))

	require.NotPanics(t, func() {
		require.NoError(t, service.VerifyContingencyBatchStatus(context.Background(), "batch-1", "mh-1", "token", 1, docs))
	})
	return calls
}

func advObsKnownContingencyIDs(docs map[string]dte.ContingencyDocument) map[string]bool {
	known := map[string]bool{}
	for _, doc := range docs {
		known[doc.ID] = true
	}
	return known
}

// TestAdversarialObservations_BatchUnknownDocumentsAndHostileObservations checks unknown documents are ignored and
// hostile observations are stored as valid JSON, one result per known document.
func TestAdversarialObservations_BatchUnknownDocumentsAndHostileObservations(t *testing.T) {
	huge := make([]string, 5000)
	for i := range huge {
		huge[i] = fmt.Sprintf("observation %d <b>", i)
	}
	long := strings.Repeat("L", 10000)
	body := `{
		"procesados":[
			{"codigoGeneracion":"GEN-OK","estado":"PROCESADO","selloRecibido":"STAMP-OK","fhProcesamiento":"14/11/2026 10:00:00","descripcionMsg":"RECIBIDO","observaciones":` + advObsJSONList(huge) + `},
			{"codigoGeneracion":"GEN-NULL","estado":"PROCESADO","selloRecibido":"STAMP-NULL","observaciones":null},
			{"codigoGeneracion":"GEN-EMPTY","estado":"PROCESADO","selloRecibido":"STAMP-EMPTY","observaciones":[]},
			{"codigoGeneracion":"GEN-UNKNOWN","estado":"PROCESADO","selloRecibido":"STAMP-X"},
			{"codigoGeneracion":"","estado":"PROCESADO","selloRecibido":"STAMP-Y"}
		],
		"rechazados":[
			{"codigoGeneracion":"GEN-BAD","estado":"RECHAZADO","codigoMsg":"095","descripcionMsg":"` + long + `","observaciones":["<script>alert(1)</script>","\"q\"\n\u0000","` + long + `"]},
			{"codigoGeneracion":"GEN-BAD-NULL","estado":"RECHAZADO","descripcionMsg":"RECHAZADO","observaciones":null},
			{"codigoGeneracion":"GEN-UNKNOWN-BAD","estado":"RECHAZADO","descripcionMsg":"RECHAZADO"}
		]}`
	docs := map[string]dte.ContingencyDocument{
		"GEN-OK":       {ID: "c-ok", DocumentID: "doc-ok"},
		"GEN-NULL":     {ID: "c-null", DocumentID: "doc-null"},
		"GEN-EMPTY":    {ID: "c-empty", DocumentID: "doc-empty"},
		"GEN-BAD":      {ID: "c-bad", DocumentID: "doc-bad"},
		"GEN-BAD-NULL": {ID: "c-bad-null", DocumentID: "doc-bad-null"},
	}

	calls := advObsRunBatch(t, body, docs)
	require.Len(t, calls, 2)
	known := advObsKnownContingencyIDs(docs)

	for _, call := range calls {
		seen := map[string]bool{}
		for _, id := range call.ids {
			assert.True(t, known[id], "unknown document %q reached UpdateBatch", id)
			assert.False(t, seen[id], "document %q repeated", id)
			seen[id] = true
			require.Contains(t, call.results, id)
		}
		assert.Len(t, call.results, len(call.ids), "one result per known document")
		assert.Len(t, call.observations, len(call.ids))
		for id, result := range call.results {
			assert.True(t, known[id])
			require.NotEmpty(t, result.HaciendaResponse, id)
			assert.True(t, json.Valid([]byte(result.HaciendaResponse)), id)
			observations := dte.ObservationsFromHaciendaResponse(result.HaciendaResponse)
			assert.NotNil(t, observations)
		}
	}

	processed, rejected := calls[0], calls[1]
	assert.Equal(t, constants.DocumentReceived, processed.status)
	assert.ElementsMatch(t, []string{"c-ok", "c-null", "c-empty"}, processed.ids)
	assert.Equal(t, huge, dte.ObservationsFromHaciendaResponse(processed.results["c-ok"].HaciendaResponse))
	assert.Equal(t, []string{}, dte.ObservationsFromHaciendaResponse(processed.results["c-null"].HaciendaResponse))
	assert.Equal(t, []string{}, dte.ObservationsFromHaciendaResponse(processed.results["c-empty"].HaciendaResponse))
	assert.Equal(t, "STAMP-NULL", processed.results["c-null"].ReceptionStamp)

	assert.Equal(t, constants.DocumentRejected, rejected.status)
	assert.ElementsMatch(t, []string{"c-bad", "c-bad-null"}, rejected.ids)
	assert.Equal(t, []string{"<script>alert(1)</script>", "\"q\"\n\x00", long},
		dte.ObservationsFromHaciendaResponse(rejected.results["c-bad"].HaciendaResponse))
	assert.Empty(t, rejected.results["c-bad"].ReceptionStamp)
	assert.Empty(t, rejected.results["c-bad"].ProcessingDate)
	assert.Equal(t, []string{}, dte.ObservationsFromHaciendaResponse(rejected.results["c-bad-null"].HaciendaResponse))
}

// TestAdversarialObservations_BatchDuplicatedGenerationCodesUpdateOnce checks a generation code repeated in the
// Hacienda answer, or present in both lists, does not repeat the document in a batch update.
func TestAdversarialObservations_BatchDuplicatedGenerationCodesUpdateOnce(t *testing.T) {
	body := `{
		"procesados":[
			{"codigoGeneracion":"GEN-DUP","estado":"PROCESADO","selloRecibido":"STAMP-1","observaciones":["first"]},
			{"codigoGeneracion":"GEN-DUP","estado":"PROCESADO","selloRecibido":"STAMP-2","observaciones":["second"]},
			{"codigoGeneracion":"GEN-BOTH","estado":"PROCESADO","selloRecibido":"STAMP-B"}
		],
		"rechazados":[
			{"codigoGeneracion":"GEN-BOTH","estado":"RECHAZADO","descripcionMsg":"RECHAZADO","observaciones":["both"]},
			{"codigoGeneracion":"GEN-REJ-DUP","estado":"RECHAZADO","descripcionMsg":"RECHAZADO"},
			{"codigoGeneracion":"GEN-REJ-DUP","estado":"RECHAZADO","descripcionMsg":"RECHAZADO"}
		]}`
	docs := map[string]dte.ContingencyDocument{
		"GEN-DUP":     {ID: "c-dup", DocumentID: "doc-dup"},
		"GEN-BOTH":    {ID: "c-both", DocumentID: "doc-both"},
		"GEN-REJ-DUP": {ID: "c-rej-dup", DocumentID: "doc-rej-dup"},
	}

	calls := advObsRunBatch(t, body, docs)
	known := advObsKnownContingencyIDs(docs)
	require.NotEmpty(t, calls)
	for _, call := range calls {
		seen := map[string]bool{}
		for _, id := range call.ids {
			assert.True(t, known[id])
			assert.False(t, seen[id], "document %q repeated in a %s update", id, call.status)
			seen[id] = true
		}
		assert.Len(t, call.results, len(seen))
	}
}
