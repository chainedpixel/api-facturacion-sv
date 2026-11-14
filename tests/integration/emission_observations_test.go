package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	coreDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/handlers"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

const emissionObservationsStamp = "2026AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"

type emissionObservationsCase struct {
	name         string
	endpoint     string
	documentType string
	requestType  interface{}
	request      func() interface{}
	build        func() (interfaces.DTEDocument, error)
	requestMap   mapper.DTEMapper
	responseMap  mapper.ResponseMapperFunc
}

func emissionObservationsCases() []emissionObservationsCase {
	factory := mapper.NewMapperFactory()
	return []emissionObservationsCase{
		{
			name:         "FE",
			endpoint:     "/invoice",
			documentType: constants.FacturaElectronica,
			requestType:  &structs.CreateInvoiceRequest{},
			request:      func() interface{} { return fixtures.CreateDefaultInvoiceRequest() },
			build:        func() (interfaces.DTEDocument, error) { return fixtures.NewDTEBuilder().BuildElectronicInvoice() },
			requestMap:   factory.CreateInvoiceMapperAdapter(),
			responseMap:  factory.GetInvoiceResponseMapper(),
		},
		{
			name:         "CCF",
			endpoint:     "/ccf",
			documentType: constants.CCFElectronico,
			requestType:  &structs.CreateCreditFiscalRequest{},
			request:      func() interface{} { return fixtures.CreateDefaultCreditFiscalRequest() },
			build:        func() (interfaces.DTEDocument, error) { return fixtures.NewDTEBuilder().BuildCreditFiscalDocument() },
			requestMap:   factory.CreateCCFMapperAdapter(),
			responseMap:  factory.GetCCFResponseMapper(),
		},
	}
}

// emitWithHaciendaResponse runs one emission through the real handler and use case and returns the recorder.
func emitWithHaciendaResponse(t *testing.T, tc emissionObservationsCase, document interfaces.DTEDocument, haciendaResponse string) *httptest.ResponseRecorder {
	t.Helper()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	authManager := mocks.NewMockAuthManager(ctrl)
	dteService := mocks.NewMockDTEService(ctrl)
	dteManager := mocks.NewMockDTEManager(ctrl)
	transmitter := mocks.NewMockBaseTransmitter(ctrl)
	contingency := mocks.NewMockContingencyManager(ctrl)
	seqManager := mocks.NewMockSequentialNumberManager(ctrl)

	authManager.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)
	dteService.EXPECT().Create(gomock.Any(), gomock.Any(), uint(1)).Return(document, nil)
	transmitter.EXPECT().RetryTransmission(gomock.Any(), gomock.Any(), "test-token", "11111111111111").
		DoAndReturn(func(_ context.Context, doc interface{}, _ string, _ string) (*transmitterModels.TransmitResult, error) {
			payload, _ := json.Marshal(doc)
			return &transmitterModels.TransmitResult{
				Status:           "PROCESADO",
				ReceptionStamp:   utils.ToStringPointer(emissionObservationsStamp),
				SignedDocument:   "eyJhbGciOiJSUzUxMiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".firma",
				ProcessingDate:   "14/11/2026 10:00:00",
				HaciendaResponse: haciendaResponse,
			}, nil
		})
	seqManager.EXPECT().ConfirmReservation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), constants.TransmissionNormal, constants.DocumentReceived, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ interface{}, _ string, _ string, evidence *coreDTE.TransmissionEvidence) error {
			assert.Equal(t, haciendaResponse, evidence.HaciendaResponse)
			return nil
		})

	useCase := dte.NewGenericDTEUseCase(authManager, dteManager, transmitter, dteService, seqManager, tc.requestMap, tc.responseMap, nil, nil)
	handler := handlers.NewGenericDTEHandler(helpers.NewContingencyHandler(contingency))
	handler.RegisterDocument(tc.endpoint, helpers.DocumentConfig{
		DocumentType:    tc.documentType,
		UseCase:         useCase,
		RequestType:     tc.requestType,
		UsesContingency: true,
	})

	body, err := json.Marshal(tc.request())
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, "/api/v1"+tc.endpoint, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "claims", &models.AuthClaims{BranchID: 1, NIT: "11111111111111"})
	req = req.WithContext(context.WithValue(ctx, "token", "test-token"))

	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1"+tc.endpoint, handler.HandleCreate).Methods(http.MethodPost)
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeEmission(t *testing.T, recorder *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	return decodeObject(t, recorder)
}

// TestEmissionObservations_ExposedAtTopLevel verifies the Hacienda observations are returned in order at the top level
// and never inside data, which stays the same as an emission without observations.
func TestEmissionObservations_ExposedAtTopLevel(t *testing.T) {
	test.TestMain(t)

	expected := []string{"Observación uno: dirección inválida", "second observation"}
	withObservations := `{"estado":"PROCESADO","observaciones":["Observación uno: dirección inválida","second observation"]}`

	for _, tc := range emissionObservationsCases() {
		t.Run(tc.name, func(t *testing.T) {
			document, err := tc.build()
			require.NoError(t, err)

			baseline := decodeEmission(t, emitWithHaciendaResponse(t, tc, document, `{"estado":"PROCESADO"}`))
			result := decodeEmission(t, emitWithHaciendaResponse(t, tc, document, withObservations))

			var observations []string
			require.NoError(t, json.Unmarshal(result["observations"], &observations))
			assert.Equal(t, expected, observations)

			var success bool
			require.NoError(t, json.Unmarshal(result["success"], &success))
			assert.True(t, success)
			assert.JSONEq(t, `"`+emissionObservationsStamp+`"`, string(result["reception_stamp"]))

			assert.JSONEq(t, string(baseline["data"]), string(result["data"]), "data is not altered by observations")

			var data map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(result["data"], &data))
			assert.NotContains(t, data, "observations")
			assert.NotContains(t, data, "observaciones")
			assert.NotContains(t, string(result["data"]), "second observation")
			assert.Contains(t, data, "firmaElectronica")
			assert.Contains(t, data, "selloRecibido")
		})
	}
}

// TestEmissionObservations_EmptyWhenNone verifies the observations key is an empty array when Hacienda sent none, the
// response is empty or it cannot be read, and the emission still succeeds.
func TestEmissionObservations_EmptyWhenNone(t *testing.T) {
	test.TestMain(t)

	responses := map[string]string{
		"without observaciones":     `{"estado":"PROCESADO"}`,
		"observaciones null":        `{"estado":"PROCESADO","observaciones":null}`,
		"observaciones empty array": `{"estado":"PROCESADO","observaciones":[]}`,
		"only empty texts":          `{"estado":"PROCESADO","observaciones":["",""]}`,
		"empty response":            ``,
		"malformed response":        `{"estado":"PROCESADO","observaciones":[`,
		"observaciones wrong type":  `{"estado":"PROCESADO","observaciones":"text"}`,
	}

	for _, tc := range emissionObservationsCases() {
		for name, haciendaResponse := range responses {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				document, err := tc.build()
				require.NoError(t, err)

				result := decodeEmission(t, emitWithHaciendaResponse(t, tc, document, haciendaResponse))

				require.Contains(t, result, "observations")
				assert.JSONEq(t, `[]`, string(result["observations"]))
				assert.Contains(t, result, "data")
			})
		}
	}
}
