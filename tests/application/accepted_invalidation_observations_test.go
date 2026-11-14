package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appdte "github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	domainDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

const acceptedInvalidationStamp = "2026INVALIDATIONSTAMP"

// acceptedInvalidationResult builds a PROCESADO transmission result whose signed document carries a JSON payload.
func acceptedInvalidationResult(haciendaResponse string) *transmitterModels.TransmitResult {
	stamp := acceptedInvalidationStamp
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"identificacion":{"version":3}}`))
	return &transmitterModels.TransmitResult{
		Status:           appdte.ReceivedStatus,
		ReceptionStamp:   &stamp,
		SignedDocument:   "header." + payload + ".signature",
		ProcessingDate:   "14/11/2026 11:00:00",
		HaciendaResponse: haciendaResponse,
	}
}

// runAcceptedInvalidation executes the use case with an accepted transmission and returns the response and the
// registered events.
func runAcceptedInvalidation(t *testing.T, result *transmitterModels.TransmitResult) (*domainDTE.EventArchiveResponse, []*domainDTE.DTEEvent, *invalidationManagerStub) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	dteManager := mocks.NewMockDTEManager(ctrl)
	transmitter := mocks.NewMockBaseTransmitter(ctrl)
	authManager := mocks.NewMockAuthManager(ctrl)
	manager := &invalidationManagerStub{}
	var registered []*domainDTE.DTEEvent

	dteManager.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), gomock.Any()).Return(storedDocument(), nil)
	authManager.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)
	transmitter.EXPECT().RetryTransmission(gomock.Any(), gomock.Any(), "token", "NIT-1").Return(result, nil)
	dteManager.EXPECT().RegisterEvent(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, event *domainDTE.DTEEvent) error {
			registered = append(registered, event)
			return nil
		})

	useCase := appdte.NewInvalidationUseCase(dteManager, manager, authManager, transmitter)
	response, err := useCase.InvalidateDocument(advContext(), invalidationRequest())
	require.NoError(t, err)
	require.NotNil(t, response)
	return response, registered, manager
}

// TestInvalidateDocument_AcceptedWithObservations verifies an accepted invalidation keeps the Hacienda response in the
// RECEIVED event and exposes its observations, in order, next to the archive.
func TestInvalidateDocument_AcceptedWithObservations(t *testing.T) {
	test.TestMain(t)

	haciendaResponse := `{"estado":"PROCESADO","codigoMsg":"001","observaciones":["Observación con tilde","second observation"]}`

	response, registered, manager := runAcceptedInvalidation(t, acceptedInvalidationResult(haciendaResponse))

	assert.Equal(t, 1, manager.invalidated)
	require.Len(t, registered, 1)
	event := registered[0]
	assert.Equal(t, constants.DocumentReceived, event.Status)
	assert.Equal(t, domainDTE.EventInvalidation, event.EventType)
	assert.Equal(t, haciendaResponse, event.HaciendaResponse)
	require.NotNil(t, event.ReceptionStamp)
	assert.Equal(t, acceptedInvalidationStamp, *event.ReceptionStamp)

	assert.Equal(t, []string{"Observación con tilde", "second observation"}, response.Observations)
	assert.Equal(t, constants.DocumentReceived, response.Status)
	require.NotNil(t, response.ReceptionStamp)
	assert.NotEmpty(t, response.Archive, "an accepted event is delivered as archive")
	assert.JSONEq(t, haciendaResponse, string(response.HaciendaResponse))

	encoded, err := json.Marshal(response)
	require.NoError(t, err)
	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.JSONEq(t, `["Observación con tilde","second observation"]`, string(decoded["observations"]))
	assert.Contains(t, decoded, "archivo_dte")
	assert.Contains(t, string(decoded["archivo_dte"]), acceptedInvalidationStamp)

	var archive map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(decoded["archivo_dte"], &archive))
	assert.NotContains(t, archive, "observations")
}

// TestInvalidateDocument_AcceptedWithoutObservations verifies the observations are an empty list, never null, when the
// accepted response has none or cannot be read.
func TestInvalidateDocument_AcceptedWithoutObservations(t *testing.T) {
	test.TestMain(t)

	responses := map[string]string{
		"without observaciones": `{"estado":"PROCESADO","codigoMsg":"001"}`,
		"observaciones null":    `{"estado":"PROCESADO","observaciones":null}`,
		"empty array":           `{"estado":"PROCESADO","observaciones":[]}`,
		"empty response":        ``,
		"malformed response":    `{"estado":`,
	}

	for name, haciendaResponse := range responses {
		t.Run(name, func(t *testing.T) {
			response, registered, _ := runAcceptedInvalidation(t, acceptedInvalidationResult(haciendaResponse))

			require.Len(t, registered, 1)
			assert.Equal(t, constants.DocumentReceived, registered[0].Status)
			assert.Equal(t, haciendaResponse, registered[0].HaciendaResponse)

			assert.NotNil(t, response.Observations)
			assert.Empty(t, response.Observations)
			assert.NotEmpty(t, response.Archive)

			encoded, err := json.Marshal(response)
			require.NoError(t, err)
			var decoded map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Contains(t, decoded, "observations")
			assert.JSONEq(t, `[]`, string(decoded["observations"]))
		})
	}
}
