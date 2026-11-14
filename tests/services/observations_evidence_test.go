package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

// obsEventRepo is an in-memory event repository used to drive DTEService.
type obsEventRepo struct {
	events  []dte.DTEEvent
	created []*dte.DTEEvent
}

func (r *obsEventRepo) Create(_ context.Context, event *dte.DTEEvent) error {
	r.created = append(r.created, event)
	return nil
}

func (r *obsEventRepo) ListByRelatedDTE(_ context.Context, _ uint, _ string) ([]dte.DTEEvent, error) {
	return r.events, nil
}

// TestObservationsFromHaciendaResponse_Table covers the parsing of the stored Hacienda response.
func TestObservationsFromHaciendaResponse_Table(t *testing.T) {
	many := make([]string, 500)
	manyJSON := make([]string, 500)
	for i := range many {
		many[i] = fmt.Sprintf("observation %d", i)
		manyJSON[i] = fmt.Sprintf("%q", many[i])
	}

	cases := []struct {
		name     string
		raw      string
		expected []string
	}{
		{"empty raw", "", []string{}},
		{"whitespace only", "   ", []string{}},
		{"invalid json", "{not json", []string{}},
		{"truncated json", `{"observaciones":["a"`, []string{}},
		{"json array root", `["a","b"]`, []string{}},
		{"json string root", `"observaciones"`, []string{}},
		{"no key", `{"estado":"PROCESADO"}`, []string{}},
		{"null value", `{"observaciones":null}`, []string{}},
		{"empty array", `{"observaciones":[]}`, []string{}},
		{"wrong type object", `{"observaciones":{"a":"b"}}`, []string{}},
		{"wrong type string", `{"observaciones":"single"}`, []string{}},
		{"wrong item type", `{"observaciones":[1,2]}`, []string{}},
		{"single value", `{"observaciones":["one"]}`, []string{"one"}},
		{"empty strings are skipped", `{"observaciones":["","a","","b",""]}`, []string{"a", "b"}},
		{"only empty strings", `{"observaciones":["",""]}`, []string{}},
		{"whitespace item is kept", `{"observaciones":[" "]}`, []string{" "}},
		{"unicode", `{"observaciones":["Descripción inválida","日本語","niño"]}`, []string{"Descripción inválida", "日本語", "niño"}},
		{"extra unknown fields", `{"estado":"RECHAZADO","foo":{"bar":[1,2,3]},"observaciones":["x"],"otro":null}`, []string{"x"}},
		{"duplicates are kept", `{"observaciones":["a","a"]}`, []string{"a", "a"}},
		{"many items", `{"observaciones":[` + strings.Join(manyJSON, ",") + `]}`, many},
		{"key matching ignores case like the json decoder", `{"Observaciones":["x"]}`, []string{"x"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := dte.ObservationsFromHaciendaResponse(tc.raw)
			require.NotNil(t, result, "the result is never nil")
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestObservationsFromHaciendaResponse_EncodesAsEmptyArray verifies the empty result serializes as [] and not null.
func TestObservationsFromHaciendaResponse_EncodesAsEmptyArray(t *testing.T) {
	raw, err := json.Marshal(dte.ObservationsFromHaciendaResponse(""))
	require.NoError(t, err)
	assert.Equal(t, "[]", string(raw))
}

const obsStoredResponse = `{"estado":"PROCESADO","observaciones":["first observation","segunda observación"]}`

func obsDocument(response string) *dte.DTEDocument {
	return &dte.DTEDocument{
		BranchID:   1,
		DocumentID: "doc-1",
		CreatedAt:  time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 5, 1, 10, 5, 0, 0, time.UTC),
		Details: &dte.DTEDetails{
			ID:               "doc-1",
			DTEType:          "01",
			ControlNumber:    "DTE-01-00000001-000000000000001",
			Transmission:     "normal",
			Status:           "RECEIVED",
			JSONData:         `{"identificacion":{"tipoDte":"01"}}`,
			HaciendaResponse: response,
		},
	}
}

func obsJWS() string {
	return jws(`{"identificacion":{"codigoGeneracion":"ABC"}}`)
}

// TestDTEService_GetByGenerationCodeConsult_Observations verifies the observations of the stored response are exposed.
func TestDTEService_GetByGenerationCodeConsult_Observations(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name     string
		response string
		expected []string
	}{
		{"with observations", obsStoredResponse, []string{"first observation", "segunda observación"}},
		{"without stored response", "", []string{}},
		{"corrupt stored response", "{corrupt", []string{}},
		{"response without observations", `{"estado":"PROCESADO"}`, []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockDTERepositoryPort(ctrl)
			repo.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), "doc-1").Return(obsDocument(tc.response), nil)

			service := dte_documents.NewDTEService(repo, nil)
			result, err := service.GetByGenerationCodeConsult(context.Background(), 1, "doc-1")
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Observations)

			raw, err := json.Marshal(result)
			require.NoError(t, err)
			var decoded map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &decoded))
			require.Contains(t, decoded, "observations", "the key is always present")
			assert.NotEqual(t, "null", string(decoded["observations"]))
		})
	}
}

// TestToEventArchiveResponse_Observations verifies observations, hacienda_response and archive of stamped and rejected events.
func TestToEventArchiveResponse_Observations(t *testing.T) {
	test.TestMain(t)

	stamp := "2026STAMP"
	empty := ""
	rejectedResponse := `{"estado":"RECHAZADO","observaciones":["bad field","invalid field"]}`
	receivedObs := []string{"first observation", "segunda observación"}

	cases := []struct {
		name            string
		event           dte.DTEEvent
		expectObs       []string
		expectArchive   bool
		expectHaciendaR bool
	}{
		{"received event keeps archive", dte.DTEEvent{SignedDocument: obsJWS(), ReceptionStamp: &stamp, HaciendaResponse: obsStoredResponse, Status: "RECEIVED"}, receivedObs, true, true},
		{"rejected event without stamp has no archive", dte.DTEEvent{SignedDocument: obsJWS(), HaciendaResponse: rejectedResponse, Status: "REJECTED"}, []string{"bad field", "invalid field"}, false, true},
		{"empty stamp pointer is not a stamp", dte.DTEEvent{SignedDocument: obsJWS(), ReceptionStamp: &empty, HaciendaResponse: rejectedResponse, Status: "REJECTED"}, []string{"bad field", "invalid field"}, false, true},
		{"stamped event without JWS has no archive", dte.DTEEvent{ReceptionStamp: &stamp, HaciendaResponse: obsStoredResponse, Status: "RECEIVED"}, receivedObs, false, true},
		{"stamped event with malformed JWS has no archive", dte.DTEEvent{SignedDocument: "not-a-jws", ReceptionStamp: &stamp, HaciendaResponse: obsStoredResponse, Status: "RECEIVED"}, receivedObs, false, true},
		{"event without response", dte.DTEEvent{SignedDocument: obsJWS(), ReceptionStamp: &stamp, Status: "RECEIVED"}, []string{}, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := dte_documents.ToEventArchiveResponse(tc.event)
			assert.Equal(t, tc.expectObs, response.Observations)
			assert.Equal(t, tc.expectArchive, len(response.Archive) > 0, "archive presence")
			assert.Equal(t, tc.expectHaciendaR, len(response.HaciendaResponse) > 0, "hacienda_response presence")

			raw, err := json.Marshal(response)
			require.NoError(t, err)
			var decoded map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &decoded))
			assert.Contains(t, decoded, "observations")
			_, hasArchive := decoded["archivo_dte"]
			assert.Equal(t, tc.expectArchive, hasArchive)
		})
	}
}

// TestToEventArchiveResponse_CorruptStoredResponseKeepsSerialization checks that an event whose stored Hacienda
// response is not valid JSON still serializes, with the response as text and empty observations.
func TestToEventArchiveResponse_CorruptStoredResponseKeepsSerialization(t *testing.T) {
	test.TestMain(t)

	response := dte_documents.ToEventArchiveResponse(dte.DTEEvent{HaciendaResponse: "{corrupt", Status: "REJECTED"})
	assert.Equal(t, []string{}, response.Observations)

	_, err := json.Marshal(response)
	assert.NoError(t, err, "a corrupt stored response must not break the serialization of the event")
}

// TestDTEService_GetEvents_RejectedEventKeepsEvidence verifies the events listing exposes rejected evidence without archive.
func TestDTEService_GetEvents_RejectedEventKeepsEvidence(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDTERepositoryPort(ctrl)
	repo.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), "doc-1").Return(obsDocument(""), nil)

	stamp := "2026STAMP"
	events := &obsEventRepo{events: []dte.DTEEvent{
		{GenerationCode: "ev-1", SignedDocument: obsJWS(), HaciendaResponse: `{"estado":"RECHAZADO","observaciones":["bad"]}`, Status: "REJECTED"},
		{GenerationCode: "ev-2", SignedDocument: obsJWS(), ReceptionStamp: &stamp, HaciendaResponse: `{"estado":"PROCESADO"}`, Status: "RECEIVED"},
	}}

	service := dte_documents.NewDTEService(repo, events)
	result, err := service.GetEvents(context.Background(), 1, "doc-1")
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, []string{"bad"}, result[0].Observations)
	assert.Empty(t, result[0].Archive)
	assert.NotEmpty(t, result[0].HaciendaResponse)
	assert.Equal(t, []string{}, result[1].Observations)
	assert.NotEmpty(t, result[1].Archive)
}
