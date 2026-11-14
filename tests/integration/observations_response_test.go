package integration_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/response"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

func decodeObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded), recorder.Body.String())
	return decoded
}

func rejectionError(observations []string) *hacienda_error.HaciendaResponseError {
	return hacienda_error.NewHaciendaError(&models.HaciendaResponse{
		Status:             hacienda_error.StatusRejected,
		MessageCode:        "095",
		DescriptionMessage: "Document rejected",
		ProcessingDate:     "14/11/2026 10:00:00",
		Observations:       observations,
	}, http.StatusBadRequest)
}

func successOptions(observations []string) *response.SuccessOptions {
	stamp := "STAMP"
	return &response.SuccessOptions{
		Ambient:        "00",
		GenerationCode: "AB123456-1234-1234-1234-AB1234567890",
		EmissionDate:   time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC),
		ReceptionStamp: &stamp,
		Observations:   observations,
	}
}

// TestResponseWriter_Success_ObservationsKey verifies the observations key is always present at the top level.
func TestResponseWriter_Success_ObservationsKey(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name         string
		observations []string
		expected     string
	}{
		{"nil becomes empty array", nil, `[]`},
		{"empty array", []string{}, `[]`},
		{"values", []string{"first", "segunda observación"}, `["first","segunda observación"]`},
		{"unicode", []string{"日本語"}, `["日本語"]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			response.NewResponseWriter().Success(recorder, http.StatusCreated, map[string]string{"k": "v"}, successOptions(tc.observations))

			assert.Equal(t, http.StatusCreated, recorder.Code)
			body := decodeObject(t, recorder)
			require.Contains(t, body, "observations")
			assert.JSONEq(t, tc.expected, string(body["observations"]))

			var data map[string]interface{}
			require.NoError(t, json.Unmarshal(body["data"], &data))
			assert.NotContains(t, data, "observations", "observations are not part of data")
			assert.Contains(t, body, "reception_stamp")
			assert.Contains(t, body, "qr_link")
		})
	}
}

// TestResponseWriter_Success_WithoutOptionsHasNoObservations keeps the plain success envelope unchanged.
func TestResponseWriter_Success_WithoutOptionsHasNoObservations(t *testing.T) {
	test.TestMain(t)

	recorder := httptest.NewRecorder()
	response.NewResponseWriter().Success(recorder, http.StatusOK, map[string]string{"k": "v"}, nil)

	body := decodeObject(t, recorder)
	assert.NotContains(t, body, "observations")
	assert.Contains(t, body, "data")
}

func decodeError(t *testing.T, recorder *httptest.ResponseRecorder) (int, map[string]json.RawMessage) {
	t.Helper()
	body := decodeObject(t, recorder)
	var errorObject map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body["error"], &errorObject))
	return recorder.Code, errorObject
}

func decodeStrings(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var values []string
	require.NoError(t, json.Unmarshal(raw, &values))
	return values
}

// TestResponseWriter_HandleError_HaciendaRejectionObservations verifies observations move from details to error.observations.
func TestResponseWriter_HandleError_HaciendaRejectionObservations(t *testing.T) {
	test.TestMain(t)

	observations := []string{"observation text one", "observación dos"}
	recorder := httptest.NewRecorder()
	response.NewResponseWriter().HandleError(recorder, rejectionError(observations))

	status, errorObject := decodeError(t, recorder)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.JSONEq(t, `"HACIENDA_095"`, string(errorObject["code"]))
	assert.Equal(t, observations, decodeStrings(t, errorObject["observations"]))

	details := decodeStrings(t, errorObject["details"])
	assert.Equal(t, []string{"State: RECHAZADO", "Processed at: 14/11/2026 10:00:00"}, details)
	for _, detail := range details {
		for _, observation := range observations {
			assert.NotContains(t, detail, observation)
		}
	}
}

// TestResponseWriter_HandleError_HaciendaRejectionWithoutObservations documents the key when Hacienda sent none.
func TestResponseWriter_HandleError_HaciendaRejectionWithoutObservations(t *testing.T) {
	test.TestMain(t)

	for name, observations := range map[string][]string{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			response.NewResponseWriter().HandleError(recorder, rejectionError(observations))

			status, errorObject := decodeError(t, recorder)
			assert.Equal(t, http.StatusBadRequest, status)
			assert.Equal(t, []string{"State: RECHAZADO", "Processed at: 14/11/2026 10:00:00"}, decodeStrings(t, errorObject["details"]))
			if raw, present := errorObject["observations"]; present {
				assert.Equal(t, []string{}, decodeStrings(t, raw), "when present it is an empty array, never null")
			}
		})
	}
}

// TestResponseWriter_HandleError_NonHaciendaErrorsHaveNoObservations verifies other errors keep their envelope.
func TestResponseWriter_HandleError_NonHaciendaErrorsHaveNoObservations(t *testing.T) {
	test.TestMain(t)

	cases := map[string]error{
		"plain error": errors.New("boom"),
		"http error":  &hacienda_error.HTTPResponseError{StatusCode: http.StatusServiceUnavailable, Method: "POST", URL: "http://mh", Body: []byte("down")},
	}

	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			response.NewResponseWriter().HandleError(recorder, err)

			_, errorObject := decodeError(t, recorder)
			assert.NotContains(t, errorObject, "observations")
		})
	}
}

// TestResponseWriter_Error_HasNoObservations verifies the generic error helper keeps its envelope.
func TestResponseWriter_Error_HasNoObservations(t *testing.T) {
	test.TestMain(t)

	recorder := httptest.NewRecorder()
	response.NewResponseWriter().Error(recorder, http.StatusBadRequest, "bad", []string{"detail"})

	_, errorObject := decodeError(t, recorder)
	assert.NotContains(t, errorObject, "observations")
}

// TestResponseWriter_HandleError_UnwrapsTransmissionError verifies the transmission wrapper does not change the response.
func TestResponseWriter_HandleError_UnwrapsTransmissionError(t *testing.T) {
	test.TestMain(t)

	observations := []string{"wrapped observation"}
	httpErr := &hacienda_error.HTTPResponseError{StatusCode: http.StatusServiceUnavailable, Method: "POST", URL: "http://mh", Body: []byte("down")}

	t.Run("hacienda rejection", func(t *testing.T) {
		wrapped := &ports.TransmissionError{Err: rejectionError(observations), SignedDocument: "jws"}
		recorder := httptest.NewRecorder()
		response.NewResponseWriter().HandleError(recorder, wrapped)

		status, errorObject := decodeError(t, recorder)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.JSONEq(t, `"HACIENDA_095"`, string(errorObject["code"]))
		assert.Equal(t, []string{"State: RECHAZADO", "Processed at: 14/11/2026 10:00:00"}, decodeStrings(t, errorObject["details"]))
		assert.Equal(t, observations, decodeStrings(t, errorObject["observations"]))
	})

	t.Run("http error keeps status and code", func(t *testing.T) {
		direct := httptest.NewRecorder()
		response.NewResponseWriter().HandleError(direct, httpErr)

		wrapped := httptest.NewRecorder()
		response.NewResponseWriter().HandleError(wrapped, &ports.TransmissionError{Err: httpErr, SignedDocument: "jws"})

		assert.Equal(t, direct.Code, wrapped.Code)
		assert.Equal(t, http.StatusServiceUnavailable, wrapped.Code)
		assert.JSONEq(t, direct.Body.String(), wrapped.Body.String())
	})

	t.Run("nested wrappers", func(t *testing.T) {
		nested := &ports.TransmissionError{Err: &ports.TransmissionError{Err: &ports.TransmissionError{Err: rejectionError(observations)}}}
		recorder := httptest.NewRecorder()
		response.NewResponseWriter().HandleError(recorder, nested)

		status, errorObject := decodeError(t, recorder)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.JSONEq(t, `"HACIENDA_095"`, string(errorObject["code"]))
		assert.Equal(t, observations, decodeStrings(t, errorObject["observations"]))
	})

	t.Run("plain inner error is a system error like the unwrapped one", func(t *testing.T) {
		direct := httptest.NewRecorder()
		response.NewResponseWriter().HandleError(direct, errors.New("boom"))

		wrapped := httptest.NewRecorder()
		response.NewResponseWriter().HandleError(wrapped, &ports.TransmissionError{Err: errors.New("boom")})

		assert.Equal(t, direct.Code, wrapped.Code)
		assert.JSONEq(t, direct.Body.String(), wrapped.Body.String())
	})
}

// TestResponseWriter_HandleError_TransmissionErrorWithoutCause checks that a transmission wrapper without inner error
// does not panic and is answered as a system error.
func TestResponseWriter_HandleError_TransmissionErrorWithoutCause(t *testing.T) {
	test.TestMain(t)

	recorder := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		response.NewResponseWriter().HandleError(recorder, &ports.TransmissionError{SignedDocument: "jws"})
	})
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
