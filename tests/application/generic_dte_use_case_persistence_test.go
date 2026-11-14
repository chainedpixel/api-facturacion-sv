package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/application/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	domainDTE "github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	transmitterModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/ordo-factus/pkg/mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

const (
	testGenerationCode = "GEN-CODE-1"
	testControlNumber  = "DTE-01-M001P001-000000000000001"
	testStamp          = "2026STAMP0001"
)

type recordingBus struct {
	mu     sync.Mutex
	events []event.Event
}

func (b *recordingBus) Publish(_ context.Context, evt event.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, evt)
}

func (b *recordingBus) Subscribe(string, event.Handler) {}

type useCaseFixture struct {
	useCase    *dte.GenericDTEUseCase
	dteManager *mocks.MockDTEManager
	seqManager *mocks.MockSequentialNumberManager
	bus        *recordingBus
	ctx        context.Context
}

func newUseCaseFixture(t *testing.T, additionalOps dte.AdditionalOperationsFunc) *useCaseFixture {
	t.Helper()
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	authManager := mocks.NewMockAuthManager(ctrl)
	dteService := mocks.NewMockDTEService(ctrl)
	dteManager := mocks.NewMockDTEManager(ctrl)
	transmitter := mocks.NewMockBaseTransmitter(ctrl)
	seqManager := mocks.NewMockSequentialNumberManager(ctrl)

	document := map[string]interface{}{
		"identificacion": map[string]interface{}{
			"tipoDte":          "01",
			"numeroControl":    testControlNumber,
			"codigoGeneracion": testGenerationCode,
		},
	}
	stamp := testStamp

	authManager.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)
	dteService.EXPECT().Create(gomock.Any(), gomock.Any(), uint(1)).Return(document, nil)
	transmitter.EXPECT().RetryTransmission(gomock.Any(), gomock.Any(), "token", "NIT-1").Return(&transmitterModels.TransmitResult{
		ReceptionStamp: &stamp,
		SignedDocument: "signed.jws.document",
		ProcessingDate: "2026-11-13T10:00:00",
	}, nil)

	requestMapper := &mapper.MapperAdapter{
		MapFunc: func(req interface{}, _ *domainDTE.IssuerDTE, _ ...interface{}) (interface{}, error) {
			return document, nil
		},
	}
	responseMapper := func(domain interface{}) interface{} { return domain }

	bus := &recordingBus{}
	useCase := dte.NewGenericDTEUseCase(authManager, dteManager, transmitter, dteService, seqManager, requestMapper, responseMapper, additionalOps, bus)

	ctx := context.WithValue(context.Background(), "claims", &models.AuthClaims{BranchID: 1, NIT: "NIT-1"})
	ctx = context.WithValue(ctx, "token", "token")

	return &useCaseFixture{useCase: useCase, dteManager: dteManager, seqManager: seqManager, bus: bus, ctx: ctx}
}

func TestGenericUseCaseRetriesPersistenceOnce(t *testing.T) {
	f := newUseCaseFixture(t, nil)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
	gomock.InOrder(
		f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db down")),
		f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
	)

	_, options, err := f.useCase.Create(f.ctx, struct{}{})

	require.NoError(t, err)
	assert.Equal(t, testStamp, *options.ReceptionStamp)
	assert.Empty(t, f.bus.events)
}

func TestGenericUseCaseReportsStampedDocumentNotPersisted(t *testing.T) {
	f := newUseCaseFixture(t, nil)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db down")).Times(2)

	_, _, err := f.useCase.Create(f.ctx, struct{}{})

	var svcErr *shared_error.ServiceError
	require.ErrorAs(t, err, &svcErr)
	assert.Equal(t, "StampedDocumentNotPersisted", svcErr.Code)
	assert.Contains(t, svcErr.Message, testGenerationCode)
	assert.Contains(t, svcErr.Message, testStamp)
	assert.False(t, strings.HasPrefix(svcErr.Message, "["), "i18n key must resolve to a message")
	assert.NotContains(t, svcErr.Message, "No se encontró documentación")

	require.Len(t, f.bus.events, 1)
	evt, ok := f.bus.events[0].(event.StampedDocumentNotPersistedEvent)
	require.True(t, ok)
	assert.Equal(t, event.EventStampedDocumentNotSaved, evt.Name())
	assert.Equal(t, testGenerationCode, evt.GenerationCode)
	assert.Equal(t, testControlNumber, evt.ControlNumber)
	assert.Equal(t, "01", evt.DTEType)
	assert.Equal(t, testStamp, evt.ReceptionStamp)
	assert.Equal(t, uint(1), evt.BranchID)
	assert.NotContains(t, evt.Payload(), "signed_document")

	contingencyHandler := helpers.NewContingencyHandler(mocks.NewMockContingencyManager(gomock.NewController(t)))
	contiType, reason := contingencyHandler.HandleContingency(f.ctx, nil, "01", err)
	assert.Nil(t, contiType)
	assert.Nil(t, reason)
}

func TestGenericUseCaseContinuesWhenConfirmReservationFails(t *testing.T) {
	f := newUseCaseFixture(t, nil)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(errors.New("confirm failed")).Times(2)
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	_, _, err := f.useCase.Create(f.ctx, struct{}{})

	require.NoError(t, err)
}

func TestGenericUseCaseWrapsUntypedAdditionalOperationsError(t *testing.T) {
	failing := func(context.Context, interface{}, uint, interface{}) error { return errors.New("invalid result type") }
	f := newUseCaseFixture(t, failing)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	_, _, err := f.useCase.Create(f.ctx, struct{}{})

	var svcErr *shared_error.ServiceError
	require.ErrorAs(t, err, &svcErr)
	assert.Equal(t, "AdditionalOperationsFailed", svcErr.Code)
	assert.False(t, strings.HasPrefix(svcErr.Message, "["))

	contingencyHandler := helpers.NewContingencyHandler(mocks.NewMockContingencyManager(gomock.NewController(t)))
	contiType, _ := contingencyHandler.HandleContingency(f.ctx, nil, "01", err)
	assert.Nil(t, contiType)
}
