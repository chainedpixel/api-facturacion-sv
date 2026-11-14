package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appdte "github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	domainDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation/invalidation_models"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/hacienda_error"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

const rejectedInvalidationJWS = "header.payload.signature"

type invalidationManagerStub struct {
	invalidated int
}

func (s *invalidationManagerStub) Validate(context.Context, uint, *invalidation_models.InvalidationDocument) error {
	return nil
}

func (s *invalidationManagerStub) ValidateStatus(context.Context, uint, structs.CreateInvalidationRequest) error {
	return nil
}

func (s *invalidationManagerStub) ValidateDependencies(context.Context, uint, *domainDTE.DTEDocument) error {
	return nil
}

func (s *invalidationManagerStub) InvalidateDocument(context.Context, uint, string) error {
	s.invalidated++
	return nil
}

func storedDocument() *domainDTE.DTEDocument {
	stamp := "2026ORIGINALSTAMP"
	return &domainDTE.DTEDocument{
		BranchID:   1,
		DocumentID: "AD54E9BB-79A3-42AE-B432-EC522C97EFB7",
		CreatedAt:  time.Now().Add(-time.Hour),
		Details: &domainDTE.DTEDetails{
			ID:             "AD54E9BB-79A3-42AE-B432-EC522C97EFB7",
			DTEType:        "01",
			ControlNumber:  "DTE-01-00000001-000000000000001",
			ReceptionStamp: &stamp,
			ProcessingDate: "14/11/2026 10:00:00",
			Status:         "RECEIVED",
			JSONData: `{"identificacion":{"fecEmi":"2026-11-14","horEmi":"09:00:00"},` +
				`"emisor":{"codActividad":"62010"},` +
				`"receptor":{"nombre":"Cliente","tipoDocumento":"13","numDocumento":"012345678","correo":"c@example.com","telefono":"22567890"},` +
				`"resumen":{"totalIva":13.0}}`,
		},
	}
}

type rejectedInvalidationFixture struct {
	useCase    *appdte.InvalidationUseCase
	dteManager *mocks.MockDTEManager
	transmit   *mocks.MockBaseTransmitter
	manager    *invalidationManagerStub
	registered []*domainDTE.DTEEvent
	registerFn func(event *domainDTE.DTEEvent) error
}

func newRejectedInvalidationFixture(t *testing.T, transmitErr error) *rejectedInvalidationFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	f := &rejectedInvalidationFixture{
		dteManager: mocks.NewMockDTEManager(ctrl),
		transmit:   mocks.NewMockBaseTransmitter(ctrl),
		manager:    &invalidationManagerStub{},
	}
	authManager := mocks.NewMockAuthManager(ctrl)

	f.dteManager.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), gomock.Any()).Return(storedDocument(), nil)
	authManager.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)
	f.transmit.EXPECT().RetryTransmission(gomock.Any(), gomock.Any(), "token", "NIT-1").Return(nil, transmitErr)
	f.dteManager.EXPECT().RegisterEvent(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, event *domainDTE.DTEEvent) error {
			f.registered = append(f.registered, event)
			if f.registerFn != nil {
				return f.registerFn(event)
			}
			return nil
		}).AnyTimes()

	f.useCase = appdte.NewInvalidationUseCase(f.dteManager, f.manager, authManager, f.transmit)
	return f
}

func invalidationRequest() structs.CreateInvalidationRequest {
	request := fixtures.CreateDefaultInvalidationRequest()
	request.ReplacementGenerationCode = nil
	return *request
}

func rejection(observations []string) *hacienda_error.HaciendaResponseError {
	return hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{
		Version:            2,
		Ambient:            "00",
		Status:             hacienda_error.StatusRejected,
		GenerationCode:     "EVENT-CODE",
		MessageCode:        "095",
		DescriptionMessage: "Rejected by Hacienda",
		ProcessingDate:     "14/11/2026 11:00:00",
		Observations:       observations,
	}, 400)
}

// TestInvalidateDocument_RejectedByHaciendaRegistersRejectedEvent verifies a RECHAZADO answer is stored as evidence and
// the original error is returned unchanged.
func TestInvalidateDocument_RejectedByHaciendaRegistersRejectedEvent(t *testing.T) {
	test.TestMain(t)

	haciendaErr := rejection([]string{"invalid field", "second observation"})
	transmitErr := &ports.TransmissionError{Err: haciendaErr, SignedDocument: rejectedInvalidationJWS}
	f := newRejectedInvalidationFixture(t, transmitErr)

	response, err := f.useCase.InvalidateDocument(advContext(), invalidationRequest())

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Same(t, transmitErr, err, "the original error is returned")
	assert.Equal(t, 0, f.manager.invalidated, "the original document is not invalidated")

	require.Len(t, f.registered, 1)
	event := f.registered[0]
	assert.Equal(t, constants.DocumentRejected, event.Status)
	assert.Equal(t, domainDTE.EventInvalidation, event.EventType)
	assert.Equal(t, rejectedInvalidationJWS, event.SignedDocument)
	assert.Nil(t, event.ReceptionStamp)
	assert.Equal(t, "14/11/2026 11:00:00", event.ProcessingDate)
	assert.Equal(t, uint(1), event.BranchID)
	require.NotNil(t, event.RelatedDTEID)
	assert.Equal(t, "AD54E9BB-79A3-42AE-B432-EC522C97EFB7", *event.RelatedDTEID)
	assert.NotEmpty(t, event.GenerationCode)
	assert.NotEmpty(t, event.JSONData)
	assert.Equal(t, []string{"invalid field", "second observation"}, domainDTE.ObservationsFromHaciendaResponse(event.HaciendaResponse))
	assert.Contains(t, event.HaciendaResponse, `"estado":"RECHAZADO"`)
	assert.Contains(t, event.HaciendaResponse, `"codigoMsg":"095"`)
}

// TestInvalidateDocument_RejectedWithoutWrapperOrObservations verifies the evidence is kept when the JWS is unknown
// or the rejection carries no observations.
func TestInvalidateDocument_RejectedWithoutWrapperOrObservations(t *testing.T) {
	test.TestMain(t)

	cases := map[string]error{
		"bare rejection without jws":             rejection([]string{"x"}),
		"rejection without observations":         &ports.TransmissionError{Err: rejection(nil), SignedDocument: rejectedInvalidationJWS},
		"rejection wrapped with fmt.Errorf":      fmt.Errorf("context: %w", &ports.TransmissionError{Err: rejection([]string{"x"}), SignedDocument: rejectedInvalidationJWS}),
		"rejection without response information": &ports.TransmissionError{Err: &hacienda_error.HaciendaResponseError{Status: hacienda_error.StatusRejected, Code: "001", Observations: []string{"only in error"}}, SignedDocument: rejectedInvalidationJWS},
	}

	for name, transmitErr := range cases {
		t.Run(name, func(t *testing.T) {
			f := newRejectedInvalidationFixture(t, transmitErr)

			_, err := f.useCase.InvalidateDocument(advContext(), invalidationRequest())
			require.Error(t, err)
			assert.Same(t, transmitErr, err)

			require.Len(t, f.registered, 1)
			event := f.registered[0]
			assert.Equal(t, constants.DocumentRejected, event.Status)
			assert.NotEmpty(t, event.HaciendaResponse)
			assert.Contains(t, event.HaciendaResponse, `"estado":"RECHAZADO"`)

			var transmissionErr *ports.TransmissionError
			if name == "bare rejection without jws" {
				assert.False(t, errors.As(err, &transmissionErr))
				assert.Empty(t, event.SignedDocument)
			} else {
				assert.Equal(t, rejectedInvalidationJWS, event.SignedDocument)
			}
		})
	}
}

// TestInvalidateDocument_OtherFailuresRegisterNothing verifies only RECHAZADO answers are stored.
func TestInvalidateDocument_OtherFailuresRegisterNothing(t *testing.T) {
	test.TestMain(t)

	notRejected := hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{
		Status: "PROCESADO", MessageCode: "001", Observations: []string{"x"},
	}, 200)
	emptyStatus := hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{MessageCode: "001"}, 400)
	lowerCase := hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{Status: "rechazado", MessageCode: "001"}, 400)

	cases := map[string]error{
		"non rejected hacienda response":    &ports.TransmissionError{Err: notRejected, SignedDocument: rejectedInvalidationJWS},
		"hacienda response without status":  &ports.TransmissionError{Err: emptyStatus, SignedDocument: rejectedInvalidationJWS},
		"status in lower case":              &ports.TransmissionError{Err: lowerCase, SignedDocument: rejectedInvalidationJWS},
		"network error":                     &ports.TransmissionError{Err: &hacienda_error.NetworkError{Operation: "transmit", OriginalError: errors.New("refused")}, SignedDocument: rejectedInvalidationJWS},
		"http response error":               &ports.TransmissionError{Err: &hacienda_error.HTTPResponseError{StatusCode: 503, Method: "POST", URL: "http://mh", Body: []byte("down")}, SignedDocument: rejectedInvalidationJWS},
		"bare http response error":          &hacienda_error.HTTPResponseError{StatusCode: 500, Method: "POST", URL: "http://mh"},
		"plain error":                       errors.New("plain"),
		"signing error":                     fmt.Errorf("%w: down", ports.ErrSigning),
		"context cancelled":                 context.Canceled,
		"transmission error with plain err": &ports.TransmissionError{Err: errors.New("plain"), SignedDocument: rejectedInvalidationJWS},
	}

	for name, transmitErr := range cases {
		t.Run(name, func(t *testing.T) {
			f := newRejectedInvalidationFixture(t, transmitErr)

			response, err := f.useCase.InvalidateDocument(advContext(), invalidationRequest())
			require.Error(t, err)
			assert.Nil(t, response)
			assert.Same(t, transmitErr, err)
			assert.Empty(t, f.registered)
		})
	}
}

// TestInvalidateDocument_RegisterFailureReturnsOriginalError verifies a storage failure does not replace the error.
func TestInvalidateDocument_RegisterFailureReturnsOriginalError(t *testing.T) {
	test.TestMain(t)

	transmitErr := &ports.TransmissionError{Err: rejection([]string{"x"}), SignedDocument: rejectedInvalidationJWS}
	f := newRejectedInvalidationFixture(t, transmitErr)
	f.registerFn = func(*domainDTE.DTEEvent) error { return errors.New("database down") }

	response, err := f.useCase.InvalidateDocument(advContext(), invalidationRequest())

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Same(t, transmitErr, err)
	assert.NotContains(t, err.Error(), "database down")
	assert.Len(t, f.registered, 1, "the registration was attempted once")
}
