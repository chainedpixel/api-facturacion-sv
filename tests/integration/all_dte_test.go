package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/application/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	transmitterModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/handlers"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/ordo-factus/pkg/mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DTETestConfig contains the configuration for the document type to test
type DTETestConfig struct {
	EndpointPath string
	DocumentType string
	RequestType  interface{}
	GetRequest   func() interface{}
	DteBuilder   func() (interfaces.DTEDocument, error)
	MapperConfig struct {
		RequestMapperAdapter mapper.DTEMapper
		ResponseMapper       mapper.ResponseMapperFunc
	}
}

func TestAllDTETypes(t *testing.T) {
	test.TestMain(t)

	factory := mapper.NewMapperFactory()

	dteConfigs := map[string]DTETestConfig{
		"Invoice": {
			EndpointPath: "/invoice",
			DocumentType: constants.FacturaElectronica,
			RequestType:  &structs.CreateInvoiceRequest{},
			GetRequest: func() interface{} {
				return fixtures.CreateDefaultInvoiceRequest()
			},
			DteBuilder: func() (interfaces.DTEDocument, error) {
				dteBuilder := fixtures.NewDTEBuilder()
				return dteBuilder.BuildElectronicInvoice()
			},
			MapperConfig: struct {
				RequestMapperAdapter mapper.DTEMapper
				ResponseMapper       mapper.ResponseMapperFunc
			}{
				RequestMapperAdapter: factory.CreateInvoiceMapperAdapter(),
				ResponseMapper:       factory.GetInvoiceResponseMapper(),
			},
		},
		"CCF": {
			EndpointPath: "/ccf",
			DocumentType: constants.CCFElectronico,
			RequestType:  &structs.CreateCreditFiscalRequest{},
			GetRequest: func() interface{} {
				return fixtures.CreateDefaultCreditFiscalRequest()
			},
			DteBuilder: func() (interfaces.DTEDocument, error) {
				dteBuilder := fixtures.NewDTEBuilder()
				return dteBuilder.BuildCreditFiscalDocument()
			},
			MapperConfig: struct {
				RequestMapperAdapter mapper.DTEMapper
				ResponseMapper       mapper.ResponseMapperFunc
			}{
				RequestMapperAdapter: factory.CreateCCFMapperAdapter(),
				ResponseMapper:       factory.GetCCFResponseMapper(),
			},
		},
		"CreditNote": {
			EndpointPath: "/creditnote",
			DocumentType: constants.NotaCreditoElectronica,
			RequestType:  &structs.CreateCreditNoteRequest{},
			GetRequest: func() interface{} {
				return fixtures.CreateDefaultCreditNoteRequest()
			},
			DteBuilder: func() (interfaces.DTEDocument, error) {
				dteBuilder := fixtures.NewDTEBuilder()
				return dteBuilder.BuildCreditNote()
			},
			MapperConfig: struct {
				RequestMapperAdapter mapper.DTEMapper
				ResponseMapper       mapper.ResponseMapperFunc
			}{
				RequestMapperAdapter: factory.CreateCreditNoteMapperAdapter(),
				ResponseMapper:       factory.GetCreditNoteResponseMapper(),
			},
		},
		"Retention": {
			EndpointPath: "/retention",
			DocumentType: constants.ComprobanteRetencionElectronico,
			RequestType:  &structs.CreateRetentionRequest{},
			GetRequest: func() interface{} {
				return fixtures.CreateMixedDocumentsRetentionRequest()
			},
			DteBuilder: func() (interfaces.DTEDocument, error) {
				dteBuilder := fixtures.NewDTEBuilder()
				return dteBuilder.BuildRetentionDocumentWithMixedItems()
			},
			MapperConfig: struct {
				RequestMapperAdapter mapper.DTEMapper
				ResponseMapper       mapper.ResponseMapperFunc
			}{
				RequestMapperAdapter: factory.CreateRetentionMapperAdapter(),
				ResponseMapper:       factory.GetRetentionResponseMapper(),
			},
		},
	}

	testCases := []struct {
		name       string
		setupMocks func(mockAuthManager *mocks.MockAuthManager, mockDTEService *mocks.MockDTEService,
			mockDTEManager *mocks.MockDTEManager, mockTransmitter *mocks.MockBaseTransmitter, mockContingency *mocks.MockContingencyManager,
			mockSeqNumberManager *mocks.MockSequentialNumberManager, dteConfig DTETestConfig)
		prepareRequest    func(dteConfig DTETestConfig) (*http.Request, error)
		expectedStatus    int
		validateResponse  func(t *testing.T, recorder *httptest.ResponseRecorder, dteConfig DTETestConfig)
		handleContingency bool
	}{
		{
			name: "Normal emission - success case",
			setupMocks: func(mockAuthManager *mocks.MockAuthManager, mockDTEService *mocks.MockDTEService,
				mockDTEManager *mocks.MockDTEManager, mockTransmitter *mocks.MockBaseTransmitter,
				mockContingency *mocks.MockContingencyManager, mockSeqNumberManager *mocks.MockSequentialNumberManager, dteConfig DTETestConfig) {

				issuer := fixtures.CreateDefaultIssuer()
				mockAuthManager.EXPECT().
					GetIssuer(gomock.Any(), uint(1)).
					Return(issuer, nil)

				mockDTE, err := dteConfig.DteBuilder()
				if err != nil {
					t.Fatalf("Error building DTE document: %v", err)
				}

				mockDTEService.EXPECT().
					Create(
						gomock.Any(),
						gomock.Any(),
						uint(1),
					).
					Return(mockDTE, nil)

				transmitResponse := &transmitterModels.TransmitResult{
					Status:         "PROCESADO",
					ReceptionStamp: utils.ToStringPointer("2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"),
				}
				mockTransmitter.EXPECT().
					RetryTransmission(
						gomock.Any(),
						gomock.Any(),
						"test-token",
						"11111111111111",
					).
					Return(transmitResponse, nil)

				mockSeqNumberManager.EXPECT().
					ConfirmReservation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				mockDTEManager.EXPECT().
					Create(
						gomock.Any(),
						gomock.Any(),
						constants.TransmissionNormal,
						constants.DocumentReceived,
						utils.ToStringPointer("2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"),
					).
					Return(nil)

				mockContingency.EXPECT().
					StoreDocumentInContingency(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).Times(0)
			},
			prepareRequest: func(dteConfig DTETestConfig) (*http.Request, error) {
				dteRequest := dteConfig.GetRequest()
				requestJSON, err := json.Marshal(dteRequest)
				if err != nil {
					return nil, err
				}

				req, err := http.NewRequest("POST", "/api/v1"+dteConfig.EndpointPath, bytes.NewBuffer(requestJSON))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Content-Type", "application/json")

				claims := &models.AuthClaims{
					BranchID: 1,
					NIT:      "11111111111111",
				}
				ctx := context.WithValue(req.Context(), "claims", claims)
				ctx = context.WithValue(ctx, "token", "test-token")
				req = req.WithContext(ctx)

				return req, nil
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, dteConfig DTETestConfig) {
				var response struct {
					Success        bool            `json:"success"`
					Data           json.RawMessage `json:"data"`
					ReceptionStamp string          `json:"reception_stamp"`
					QRLink         string          `json:"qr_link"`
				}

				err := json.NewDecoder(recorder.Body).Decode(&response)
				require.NoError(t, err)

				assert.True(t, response.Success)
				assert.NotEmpty(t, response.ReceptionStamp)
				assert.NotEmpty(t, response.QRLink)
				assert.Equal(t, "2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM", response.ReceptionStamp)

				var identification struct {
					Identificacion struct {
						Version       int    `json:"version"`
						Ambiente      string `json:"ambiente"`
						TipoDte       string `json:"tipoDte"`
						TipoOperacion int    `json:"tipoOperacion"`
						TipoModelo    int    `json:"tipoModelo"`
						NumeroControl string `json:"numeroControl"`
					} `json:"identificacion"`
				}

				err = json.Unmarshal(response.Data, &identification)
				require.NoError(t, err)

				assert.Equal(t, 1, identification.Identificacion.Version)
				assert.Equal(t, constants.Testing, identification.Identificacion.Ambiente)
				assert.Equal(t, dteConfig.DocumentType, identification.Identificacion.TipoDte)
				assert.Equal(t, constants.TransmisionNormal, identification.Identificacion.TipoOperacion)
				assert.Equal(t, constants.ModeloFacturacionPrevio, identification.Identificacion.TipoModelo)
				assert.NotEmpty(t, identification.Identificacion.NumeroControl)
			},
			handleContingency: false,
		},
		{
			name: "Contingency emission - success case",
			setupMocks: func(mockAuthManager *mocks.MockAuthManager, mockDTEService *mocks.MockDTEService,
				mockDTEManager *mocks.MockDTEManager, mockTransmitter *mocks.MockBaseTransmitter,
				mockContingency *mocks.MockContingencyManager, mockSeqNumberManager *mocks.MockSequentialNumberManager, dteConfig DTETestConfig) {

				issuer := fixtures.CreateDefaultIssuer()
				mockAuthManager.EXPECT().
					GetIssuer(gomock.Any(), uint(1)).
					Return(issuer, nil)

				mockDTE, err := dteConfig.DteBuilder()
				if err != nil {
					t.Fatalf("Error building DTE document: %v", err)
				}

				mockDTEService.EXPECT().
					Create(
						gomock.Any(),
						gomock.Any(),
						uint(1),
					).
					Return(mockDTE, nil)

				mockTransmitter.EXPECT().
					RetryTransmission(
						gomock.Any(),
						gomock.Any(),
						"test-token",
						"11111111111111",
					).
					Return(nil, &hacienda_error.HTTPResponseError{
						StatusCode: http.StatusServiceUnavailable,
						Body:       []byte("Forced contingency - service unavailable"),
						URL:        config.MHPaths.ReceptionURL,
						Method:     "POST",
					}).
					AnyTimes()

				mockContingency.EXPECT().
					StoreDocumentInContingency(
						gomock.Any(),
						gomock.Any(),
						dteConfig.DocumentType,
						int8(constants.NoDisponibilidadMH),
						constants.ContingencyReasons[constants.NoDisponibilidadMH],
					).Return(nil)
			},
			prepareRequest: func(dteConfig DTETestConfig) (*http.Request, error) {
				dteRequest := dteConfig.GetRequest()
				requestJSON, err := json.Marshal(dteRequest)
				if err != nil {
					return nil, err
				}

				req, err := http.NewRequest("POST", "/api/v1"+dteConfig.EndpointPath, bytes.NewBuffer(requestJSON))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Content-Type", "application/json")

				claims := &models.AuthClaims{
					BranchID: 1,
					NIT:      "11111111111111",
				}
				ctx := context.WithValue(req.Context(), "claims", claims)
				ctx = context.WithValue(ctx, "token", "test-token")
				req = req.WithContext(ctx)

				return req, nil
			},
			expectedStatus: http.StatusCreated,
			validateResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, dteConfig DTETestConfig) {
				var response struct {
					Success        bool            `json:"success"`
					Data           json.RawMessage `json:"data"`
					ReceptionStamp *string         `json:"reception_stamp"`
					QRLink         string          `json:"qr_link"`
				}

				err := json.NewDecoder(recorder.Body).Decode(&response)
				require.NoError(t, err)

				assert.True(t, response.Success)
				assert.Nil(t, response.ReceptionStamp)

				var identification struct {
					Identificacion struct {
						Version          int    `json:"version"`
						Ambiente         string `json:"ambiente"`
						TipoDte          string `json:"tipoDte"`
						TipoOperacion    int    `json:"tipoOperacion"`
						TipoModelo       int    `json:"tipoModelo"`
						NumeroControl    string `json:"numeroControl"`
						TipoContingencia int    `json:"tipoContingencia"`
						MotivoContin     string `json:"motivoContin"`
					} `json:"identificacion"`
				}

				err = json.Unmarshal(response.Data, &identification)
				require.NoError(t, err)

				assert.Equal(t, dteConfig.DocumentType, identification.Identificacion.TipoDte)
				assert.Equal(t, constants.TransmisionContingencia, identification.Identificacion.TipoOperacion)
				assert.Equal(t, constants.ModeloFacturacionDiferido, identification.Identificacion.TipoModelo)
				assert.Equal(t, constants.NoDisponibilidadMH, identification.Identificacion.TipoContingencia)
				assert.Equal(t, constants.ContingencyReasons[constants.NoDisponibilidadMH], identification.Identificacion.MotivoContin)
			},
			handleContingency: true,
		},
		{
			name: "Validation error - error case",
			setupMocks: func(mockAuthManager *mocks.MockAuthManager, mockDTEService *mocks.MockDTEService,
				mockDTEManager *mocks.MockDTEManager, mockTransmitter *mocks.MockBaseTransmitter,
				mockContingency *mocks.MockContingencyManager, mockSeqNumberManager *mocks.MockSequentialNumberManager, dteConfig DTETestConfig) {

				issuer := fixtures.CreateDefaultIssuer()
				mockAuthManager.EXPECT().
					GetIssuer(gomock.Any(), uint(1)).
					Return(issuer, nil)

				mockDTEService.EXPECT().
					Create(
						gomock.Any(),
						gomock.Any(),
						uint(1),
					).
					Return(nil, dte_errors.NewValidationError("RequiredField", "Request->Receiver"))

				mockContingency.EXPECT().
					StoreDocumentInContingency(
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
						gomock.Any(),
					).Times(0)
			},
			prepareRequest: func(dteConfig DTETestConfig) (*http.Request, error) {
				dteRequest := dteConfig.GetRequest()
				doInvalidAmount(dteRequest)

				requestJSON, err := json.Marshal(dteRequest)
				if err != nil {
					return nil, err
				}

				req, err := http.NewRequest("POST", "/api/v1"+dteConfig.EndpointPath, bytes.NewBuffer(requestJSON))
				if err != nil {
					return nil, err
				}
				req.Header.Set("Content-Type", "application/json")

				claims := &models.AuthClaims{
					BranchID: 1,
					NIT:      "11111111111111",
				}
				ctx := context.WithValue(req.Context(), "claims", claims)
				ctx = context.WithValue(ctx, "token", "test-token")
				req = req.WithContext(ctx)

				return req, nil
			},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, recorder *httptest.ResponseRecorder, dteConfig DTETestConfig) {
				var response struct {
					Success bool            `json:"success"`
					Error   json.RawMessage `json:"error"`
				}

				err := json.NewDecoder(recorder.Body).Decode(&response)
				require.NoError(t, err)

				assert.False(t, response.Success)
				assert.NotNil(t, response.Error)

				errorStr := string(response.Error)
				assert.Contains(t, errorStr, "required")
			},
			handleContingency: false,
		},
	}

	for dteName, dteConfig := range dteConfigs {
		t.Run(dteName, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					defer ctrl.Finish()

					mockAuthManager := mocks.NewMockAuthManager(ctrl)
					mockDTEService := mocks.NewMockDTEService(ctrl)
					mockDTEManager := mocks.NewMockDTEManager(ctrl)
					mockTransmitter := mocks.NewMockBaseTransmitter(ctrl)
					mockContingency := mocks.NewMockContingencyManager(ctrl)
					mockSeqNumberManager := mocks.NewMockSequentialNumberManager(ctrl)

					tc.setupMocks(mockAuthManager, mockDTEService, mockDTEManager, mockTransmitter, mockContingency, mockSeqNumberManager, dteConfig)

					var additionalOps dte.AdditionalOperationsFunc = nil
					genericUseCase := dte.NewGenericDTEUseCase(
						mockAuthManager,
						mockDTEManager,
						mockTransmitter,
						mockDTEService,
						mockSeqNumberManager,
						dteConfig.MapperConfig.RequestMapperAdapter,
						dteConfig.MapperConfig.ResponseMapper,
						additionalOps,
					)

					contingencyHandler := helpers.NewContingencyHandler(mockContingency)
					genericHandler := handlers.NewGenericDTEHandler(contingencyHandler)

					genericHandler.RegisterDocument(dteConfig.EndpointPath, helpers.DocumentConfig{
						DocumentType:    dteConfig.DocumentType,
						UseCase:         genericUseCase,
						RequestType:     dteConfig.RequestType,
						UsesContingency: true,
					})

					req, err := tc.prepareRequest(dteConfig)
					require.NoError(t, err)

					recorder := httptest.NewRecorder()
					router := mux.NewRouter()
					router.HandleFunc("/api/v1"+dteConfig.EndpointPath, genericHandler.HandleCreate).Methods("POST")
					router.ServeHTTP(recorder, req)

					assert.Equal(t, tc.expectedStatus, recorder.Code)

					tc.validateResponse(t, recorder, dteConfig)
				})
			}
		})
	}
}

// doInvalidAmount modifies the amount of a DTE to make it invalid at the domain level
func doInvalidAmount(
	dteRequest interface{},
) {
	invalidValue := 40.0
	switch req := dteRequest.(type) {
	case *structs.CreateInvoiceRequest:
		req.Items[0].TaxedSale = invalidValue
	case *structs.CreateCreditFiscalRequest:
		req.Items[0].TaxedSale = invalidValue
	case *structs.CreateCreditNoteRequest:
		req.Items[0].TaxedSale = invalidValue
	case *structs.CreateRetentionRequest:
		req.Items[0].TaxedAmount = &invalidValue
	}
}
