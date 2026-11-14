package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/response"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

func advObsForcedConfig(t *testing.T, enabled bool, ambient string) {
	t.Helper()
	test.TestMain(t)
	previousEnabled, previousAmbient := config.Server.ForceContingency, config.Server.AmbientCode
	config.Server.ForceContingency = enabled
	config.Server.AmbientCode = ambient
	t.Cleanup(func() {
		config.Server.ForceContingency = previousEnabled
		config.Server.AmbientCode = previousAmbient
	})
}

// TestAdversarialObservations_ForcedBranchConcurrentRequestsKeepTheirOwnType runs the real header helper and the real
// transmitter forced branch from many goroutines and checks each result matches its own header.
func TestAdversarialObservations_ForcedBranchConcurrentRequestsKeepTheirOwnType(t *testing.T) {
	advObsForcedConfig(t, true, "00")
	mh := transmitter.NewMHTransmitter(nil, nil)

	headers := []string{"5", "1", " 5 ", "abc", "", "3", "5\t", "-0", "+5", "05"}
	wantStatus := func(header string) int {
		switch strings.TrimSpace(header) {
		case "5", "+5", "05":
			return http.StatusInternalServerError
		}
		return http.StatusServiceUnavailable
	}

	var wg sync.WaitGroup
	failures := make(chan string, 6000)
	for i := 0; i < 3000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			header := headers[i%len(headers)]
			req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice", nil)
			req.Header.Set(helpers.ForcedContingencyHeader, header)
			ctx := helpers.RequestContextWithForcedContingency(req)

			result, err := mh.Transmit(ctx, struct{}{}, "jws", "token")
			var httpErr *hacienda_error.HTTPResponseError
			if result != nil || !errors.As(err, &httpErr) {
				failures <- fmt.Sprintf("header %q: unexpected result %v err %v", header, result, err)
				return
			}
			if httpErr.StatusCode != wantStatus(header) {
				failures <- fmt.Sprintf("header %q: status %d, want %d (forced type %d)", header, httpErr.StatusCode,
					wantStatus(header), utils.ForcedContingencyType(ctx))
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}

// TestAdversarialObservations_TransmitterNeverSimulatesOutsideTestAmbient checks that the forced branch is closed when
// the ambient is production or forced contingency is disabled, whatever the context asks for.
func TestAdversarialObservations_TransmitterNeverSimulatesOutsideTestAmbient(t *testing.T) {
	mh := transmitter.NewMHTransmitter(nil, nil)
	cases := []struct {
		name    string
		enabled bool
		ambient string
	}{
		{"production ambient with flag", true, "01"},
		{"production ambient without flag", false, "01"},
		{"test ambient without flag", false, "00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			advObsForcedConfig(t, tc.enabled, tc.ambient)
			for _, forced := range []int{0, 1, 5, 99} {
				ctx := utils.WithForcedContingencyType(context.Background(), forced)
				var result *models.TransmitResult
				var err error
				require.NotPanics(t, func() { result, err = mh.Transmit(ctx, struct{}{}, "jws", "token") })
				assert.Nil(t, result)
				require.Error(t, err)
				var httpErr *hacienda_error.HTTPResponseError
				assert.False(t, errors.As(err, &httpErr), "forced type %d must not simulate: %v", forced, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set(helpers.ForcedContingencyHeader, "5")
			assert.Zero(t, utils.ForcedContingencyType(helpers.RequestContextWithForcedContingency(req)))
		})
	}
}

// TestAdversarialObservations_TransmitterForcedBranchIgnoresUnsupportedContextTypes checks that unsupported forced
// types injected directly in the context fall back to the unavailable simulation.
func TestAdversarialObservations_TransmitterForcedBranchIgnoresUnsupportedContextTypes(t *testing.T) {
	advObsForcedConfig(t, true, "00")
	mh := transmitter.NewMHTransmitter(nil, nil)
	for _, forced := range []int{-1, 0, 2, 3, 4, 6, 99} {
		ctx := utils.WithForcedContingencyType(context.Background(), forced)
		_, err := mh.Transmit(ctx, struct{}{}, "jws", "token")
		var httpErr *hacienda_error.HTTPResponseError
		require.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusServiceUnavailable, httpErr.StatusCode, "type %d", forced)
	}
	_, err := mh.Transmit(context.TODO(), struct{}{}, "jws", "token")
	var httpErr *hacienda_error.HTTPResponseError
	require.True(t, errors.As(err, &httpErr))
	assert.Equal(t, http.StatusServiceUnavailable, httpErr.StatusCode)
}

func advObsSuccessBody(t *testing.T, observations []string) (int, map[string]json.RawMessage, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	response.NewResponseWriter().Success(recorder, http.StatusCreated, map[string]string{"k": "v"}, successOptions(observations))
	require.True(t, json.Valid(recorder.Body.Bytes()), recorder.Body.String())
	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded))
	return recorder.Code, decoded, recorder.Body.String()
}

// TestAdversarialObservations_SuccessWriterEscapesHostileTexts checks hostile observation texts round trip as valid
// JSON, in the same order, never null.
func TestAdversarialObservations_SuccessWriterEscapesHostileTexts(t *testing.T) {
	test.TestMain(t)

	huge := make([]string, 5000)
	for i := range huge {
		huge[i] = fmt.Sprintf("observation %d <b>\"q\"</b>", i)
	}
	cases := map[string][]string{
		"html and script":    {"<script>alert('x')</script>", "<img src=x onerror=alert(1)>", "&amp; &lt;"},
		"quotes and slashes": {`"quoted"`, `back\slash`, `\"`, `\\`, `\u0000`, `"}],"success":false,"x":["`},
		"newlines":           {"line1\nline2", "tab\there", "cr\rhere", "\r\n"},
		"control characters": {"\x00\x01\x02\x07\x1b\x7f", "nul\x00inside"},
		"line separators":    {"  ", "emoji \U0001F600", "japanese 日本語"},
		"invalid utf8":       {"bad \xff\xfe bytes", "\xc3\x28"},
		"10k characters":     {strings.Repeat("x", 10000), strings.Repeat("\"", 10000)},
		"5000 items":         huge,
		"empty strings kept": {"", "", "a"},
	}
	for name, observations := range cases {
		t.Run(name, func(t *testing.T) {
			code, body, raw := advObsSuccessBody(t, observations)
			assert.Equal(t, http.StatusCreated, code)
			require.Contains(t, body, "observations")
			assert.NotEqual(t, "null", string(body["observations"]))
			assert.NotContains(t, raw, "<script>", "html must be escaped")

			var decoded []string
			require.NoError(t, json.Unmarshal(body["observations"], &decoded))
			require.Len(t, decoded, len(observations))
			if name != "invalid utf8" {
				assert.Equal(t, observations, decoded, "order and content")
			}

			var data map[string]string
			require.NoError(t, json.Unmarshal(body["data"], &data))
			assert.Equal(t, map[string]string{"k": "v"}, data)
		})
	}
}

// TestAdversarialObservations_ErrorWriterTransmissionWrappers checks the status, code and observations key for every
// shape of wrapped error.
func TestAdversarialObservations_ErrorWriterTransmissionWrappers(t *testing.T) {
	test.TestMain(t)

	hostile := []string{"<script>x</script>", "\"quoted\"\n", strings.Repeat("y", 10000)}
	cases := []struct {
		name            string
		err             error
		wantStatus      int
		wantCode        string
		wantObservation *[]string
	}{
		{
			name:            "wrapper around rejection with nil observations",
			err:             &ports.TransmissionError{Err: rejectionError(nil), SignedDocument: "jws"},
			wantStatus:      http.StatusBadRequest,
			wantCode:        "HACIENDA_095",
			wantObservation: &[]string{},
		},
		{
			name:            "wrapper around rejection with empty observations",
			err:             &ports.TransmissionError{Err: rejectionError([]string{}), SignedDocument: "jws"},
			wantStatus:      http.StatusBadRequest,
			wantCode:        "HACIENDA_095",
			wantObservation: &[]string{},
		},
		{
			name:            "wrapper around rejection with hostile observations",
			err:             &ports.TransmissionError{Err: rejectionError(hostile), SignedDocument: ""},
			wantStatus:      http.StatusBadRequest,
			wantCode:        "HACIENDA_095",
			wantObservation: &hostile,
		},
		{
			name: "triple nested wrappers",
			err: &ports.TransmissionError{Err: &ports.TransmissionError{
				Err: &ports.TransmissionError{Err: rejectionError([]string{"deep"})}}},
			wantStatus:      http.StatusBadRequest,
			wantCode:        "HACIENDA_095",
			wantObservation: &[]string{"deep"},
		},
		{
			name:       "wrapper around http error",
			err:        &ports.TransmissionError{Err: &hacienda_error.HTTPResponseError{StatusCode: 503, Method: "POST", URL: "http://mh", Body: []byte("down")}, SignedDocument: "jws"},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "HACIENDA_503",
		},
		{
			name:       "wrapper around http error 500",
			err:        &ports.TransmissionError{Err: &hacienda_error.HTTPResponseError{StatusCode: 500, Method: "POST", URL: "http://mh"}},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "HACIENDA_500",
		},
		{
			name:       "wrapper around plain error",
			err:        &ports.TransmissionError{Err: errors.New("plain"), SignedDocument: "jws"},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "SYSTEM_ERROR",
		},
		{
			name:       "wrapper with nil inner error",
			err:        &ports.TransmissionError{SignedDocument: "jws"},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "SYSTEM_ERROR",
		},
		{
			name:       "wrapper with nil inner error nested",
			err:        &ports.TransmissionError{Err: &ports.TransmissionError{}},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "SYSTEM_ERROR",
		},
		{
			name:            "bare rejection",
			err:             rejectionError(nil),
			wantStatus:      http.StatusBadRequest,
			wantCode:        "HACIENDA_095",
			wantObservation: &[]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			require.NotPanics(t, func() { response.NewResponseWriter().HandleError(recorder, tc.err) })
			assert.Equal(t, tc.wantStatus, recorder.Code)
			require.True(t, json.Valid(recorder.Body.Bytes()), recorder.Body.String())

			var decoded struct {
				Success bool `json:"success"`
				Error   struct {
					Code         string    `json:"code"`
					Details      []string  `json:"details"`
					Observations *[]string `json:"observations"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded))
			assert.False(t, decoded.Success)
			assert.Equal(t, tc.wantCode, decoded.Error.Code)

			if tc.wantObservation == nil {
				assert.Nil(t, decoded.Error.Observations, "only Hacienda rejections carry observations")
				return
			}
			require.NotNil(t, decoded.Error.Observations)
			assert.Equal(t, *tc.wantObservation, *decoded.Error.Observations)
			require.Len(t, decoded.Error.Details, 2)
			assert.True(t, strings.HasPrefix(decoded.Error.Details[0], "State:"))
			assert.True(t, strings.HasPrefix(decoded.Error.Details[1], "Processed at:"))
			for _, detail := range decoded.Error.Details {
				for _, observation := range *tc.wantObservation {
					assert.NotContains(t, detail, observation, "observations are not repeated in details")
				}
			}
		})
	}
}

// TestAdversarialObservations_TransmissionErrorMessages checks the message of the wrapper for nil and nested errors.
func TestAdversarialObservations_TransmissionErrorMessages(t *testing.T) {
	test.TestMain(t)

	empty := &ports.TransmissionError{}
	assert.NotPanics(t, func() { _ = empty.Error() })
	assert.Equal(t, "transmission failed", empty.Error())
	assert.Nil(t, empty.Unwrap())
	assert.False(t, errors.Is(empty, errors.New("x")))

	inner := &ports.TransmissionError{Err: errors.New("deep")}
	outer := &ports.TransmissionError{Err: inner}
	assert.Equal(t, "deep", outer.Error())

	wrapped := &ports.TransmissionError{Err: &hacienda_error.HaciendaResponseError{}}
	assert.NotPanics(t, func() { _ = wrapped.Error() })

	rejection := hacienda_error.NewHaciendaError(&models.HaciendaResponse{}, 400)
	recorder := httptest.NewRecorder()
	assert.NotPanics(t, func() {
		response.NewResponseWriter().HandleError(recorder, &ports.TransmissionError{Err: rejection})
	})
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
