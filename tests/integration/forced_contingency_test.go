package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/auth/models"
	coreDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/contingency"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/handlers"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

// runForcedContingencyRequest sends one create request through the real handler, the real base transmitter and the
// real Hacienda transmitter with forced contingency enabled, using the given header value when it is not nil.
func runForcedContingencyRequest(t *testing.T, docType contingencyDocType, header *string) *contingencyOutcome {
	t.Helper()
	test.TestMain(t)
	previousEnabled, previousAmbient := config.Server.ForceContingency, config.Server.AmbientCode
	config.Server.ForceContingency = true
	config.Server.AmbientCode = "00"
	t.Cleanup(func() {
		config.Server.ForceContingency = previousEnabled
		config.Server.AmbientCode = previousAmbient
	})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	outcome := &contingencyOutcome{signer: &contingencyFakeSigner{}}

	mockAuth := mocks.NewMockAuthManager(ctrl)
	mockAuth.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)

	mockDTE, err := docType.dte()
	require.NoError(t, err)
	mockDTEService := mocks.NewMockDTEService(ctrl)
	mockDTEService.EXPECT().Create(gomock.Any(), gomock.Any(), uint(1)).Return(mockDTE, nil)

	mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
	mockSeq.EXPECT().ReleaseReservation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string, string, string, uint) error {
			outcome.releaseCalls++
			return nil
		}).AnyTimes()
	mockSeq.EXPECT().MarkReservationAsContingencyByControlNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	mockDTEManager := mocks.NewMockDTEManager(ctrl)
	mockDTEManager.EXPECT().
		Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, gomock.Any()).
		DoAndReturn(func(_ context.Context, document interface{}, _ string, _ string, given *coreDTE.TransmissionEvidence) error {
			outcome.dteCreateCalls++
			outcome.storedDocument = document
			outcome.evidence = given
			return nil
		}).AnyTimes()

	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, *coreDTE.ContingencyDocument) error {
			outcome.repoCreateCalls++
			return nil
		}).AnyTimes()

	manager := contingency.NewContingencyManager(nil, mockDTEManager, mockRepo, nil, nil, nil, outcome.signer, nil,
		mocks.NewMockContingencyEventSender(ctrl), mockSeq, nil, nil)

	baseTransmitter := dte.NewBaseTransmitter(transmitter.NewMHTransmitter(nil, nil), outcome.signer)
	requestMapper, responseMapper := docType.mapper()
	useCase := dte.NewGenericDTEUseCase(mockAuth, mockDTEManager, baseTransmitter, mockDTEService, mockSeq,
		requestMapper, responseMapper, nil, nil)

	handler := handlers.NewGenericDTEHandler(helpers.NewContingencyHandler(manager))
	handler.RegisterDocument(docType.endpoint, helpers.DocumentConfig{
		DocumentType:    docType.documentType,
		UseCase:         useCase,
		RequestType:     docType.requestType,
		UsesContingency: true,
	})

	body, err := json.Marshal(docType.request())
	require.NoError(t, err)
	req, err := http.NewRequest("POST", "/api/v1"+docType.endpoint, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if header != nil {
		req.Header.Set(helpers.ForcedContingencyHeader, *header)
	}
	ctx := context.WithValue(req.Context(), "claims", &models.AuthClaims{BranchID: 1, NIT: "11111111111111"})
	ctx = context.WithValue(ctx, "token", "test-token")
	req = req.WithContext(ctx)

	outcome.recorder = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1"+docType.endpoint, handler.HandleCreate).Methods("POST")
	router.ServeHTTP(outcome.recorder, req)
	return outcome
}

func stringPointer(value string) *string { return &value }

// TestForcedContingencyHeaderSelectsContingencyType checks, through the real handler and the real transmitter, the
// contingency type and reason delivered for each header value.
func TestForcedContingencyHeaderSelectsContingencyType(t *testing.T) {
	cases := []struct {
		name       string
		header     *string
		wantType   int
		wantReason bool
	}{
		{"header 5", stringPointer("5"), constants.OtroMotivo, true},
		{"header 5 with spaces", stringPointer(" 5 "), constants.OtroMotivo, true},
		{"header 1", stringPointer("1"), constants.NoDisponibilidadMH, false},
		{"no header", nil, constants.NoDisponibilidadMH, false},
		{"unsupported type 3", stringPointer("3"), constants.NoDisponibilidadMH, false},
		{"invalid header", stringPointer("abc"), constants.NoDisponibilidadMH, false},
	}

	for _, docType := range contingencyDocTypes() {
		for _, tc := range cases {
			t.Run(docType.name+"/"+tc.name, func(t *testing.T) {
				outcome := runForcedContingencyRequest(t, docType, tc.header)

				require.Equal(t, http.StatusCreated, outcome.recorder.Code, outcome.recorder.Body.String())
				envelope := decodeContingencyEnvelope(t, outcome)
				assert.True(t, envelope.Success)

				identification := contingencyIdentification(t, envelope.Data)
				assert.EqualValues(t, constants.ModeloFacturacionDiferido, identification["tipoModelo"])
				assert.EqualValues(t, constants.TransmisionContingencia, identification["tipoOperacion"])
				assert.EqualValues(t, tc.wantType, identification["tipoContingencia"])
				if tc.wantReason {
					reason, ok := identification["motivoContin"].(string)
					require.True(t, ok, "type 5 must carry a reason")
					assert.NotEmpty(t, strings.TrimSpace(reason))
				} else {
					assert.Nil(t, identification["motivoContin"])
				}

				signature, _ := envelope.Data["firmaElectronica"].(string)
				require.NotEmpty(t, signature)
				assertFieldByField(t, contingencyWithoutSignature(envelope.Data), contingencyJWSPayload(t, signature))

				require.Equal(t, 1, outcome.dteCreateCalls)
				require.Equal(t, 1, outcome.repoCreateCalls)
				require.NotNil(t, outcome.evidence)
				assert.Equal(t, signature, outcome.evidence.SignedDocument)
				assert.Nil(t, outcome.evidence.ReceptionStamp)
			})
		}
	}
}
