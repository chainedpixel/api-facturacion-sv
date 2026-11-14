package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/application/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	domainDTE "github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	transmitterModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/ordo-factus/pkg/mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

type advFixture struct {
	useCase    *dte.GenericDTEUseCase
	dteManager *mocks.MockDTEManager
	seqManager *mocks.MockSequentialNumberManager
	ctrl       *gomock.Controller
}

func newAdvFixture(t *testing.T, ops dte.AdditionalOperationsFunc, bus event.Bus, withStamp bool) *advFixture {
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
	result := &transmitterModels.TransmitResult{SignedDocument: "signed.jws.document", ProcessingDate: "2026-11-13T10:00:00"}
	if withStamp {
		result.ReceptionStamp = &stamp
	}

	authManager.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)
	dteService.EXPECT().Create(gomock.Any(), gomock.Any(), uint(1)).Return(document, nil)
	transmitter.EXPECT().RetryTransmission(gomock.Any(), gomock.Any(), "token", "NIT-1").Return(result, nil)

	requestMapper := &mapper.MapperAdapter{
		MapFunc: func(interface{}, *domainDTE.IssuerDTE, ...interface{}) (interface{}, error) { return document, nil },
	}
	responseMapper := func(domain interface{}) interface{} { return domain }

	useCase := dte.NewGenericDTEUseCase(authManager, dteManager, transmitter, dteService, seqManager, requestMapper, responseMapper, ops, bus)
	return &advFixture{useCase: useCase, dteManager: dteManager, seqManager: seqManager, ctrl: ctrl}
}

func advContext() context.Context {
	ctx := context.WithValue(context.Background(), "claims", &models.AuthClaims{BranchID: 1, NIT: "NIT-1"})
	return context.WithValue(ctx, "token", "token")
}

func requireServiceError(t *testing.T, err error, code string) *shared_error.ServiceError {
	t.Helper()
	var svcErr *shared_error.ServiceError
	require.ErrorAs(t, err, &svcErr)
	assert.Equal(t, code, svcErr.Code)
	assert.False(t, strings.HasPrefix(svcErr.Message, "["), "i18n key must resolve: %s", svcErr.Message)
	assert.NotContains(t, svcErr.Message, "No se encontró documentación")
	assert.NotContains(t, svcErr.Message, "No documentation found")
	assert.NotContains(t, svcErr.Message, "%!")
	return svcErr
}

// TestAdversarialPersistFailureWithEveryErrorShape verifies typed and untyped Create failures are all reported as stamped-not-persisted.
func TestAdversarialPersistFailureWithEveryErrorShape(t *testing.T) {
	test.TestMain(t)

	causes := map[string]error{
		"untyped":    errors.New("connection reset"),
		"validation": dte_errors.NewValidationError("InvalidAmount", "x"),
		"service":    shared_error.NewGeneralServiceError("Repo", "Create", "duplicate", nil),
		"dte":        dte_errors.NewDTEErrorSimple("InvalidItemSaleAmount", 1, 2.0, 3.0),
	}

	for name, cause := range causes {
		t.Run(name, func(t *testing.T) {
			bus := &recordingBus{}
			f := newAdvFixture(t, nil, bus, true)
			f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
			f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(cause).Times(2)

			_, _, err := f.useCase.Create(advContext(), struct{}{})

			svcErr := requireServiceError(t, err, "StampedDocumentNotPersisted")
			assert.Contains(t, svcErr.Message, testGenerationCode)
			assert.Contains(t, svcErr.Message, testStamp)
			require.Len(t, bus.events, 1)

			handler := helpers.NewContingencyHandler(mocks.NewMockContingencyManager(gomock.NewController(t)))
			contiType, _ := handler.HandleContingency(advContext(), nil, "01", err)
			assert.Nil(t, contiType, "a stamped document must never be sent to contingency")
		})
	}
}

// TestAdversarialPersistFailureWithoutBusOrStamp verifies a nil bus and a missing reception stamp do not panic.
func TestAdversarialPersistFailureWithoutBusOrStamp(t *testing.T) {
	f := newAdvFixture(t, nil, nil, false)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db down")).Times(2)

	assert.NotPanics(t, func() {
		_, _, err := f.useCase.Create(advContext(), struct{}{})
		requireServiceError(t, err, "StampedDocumentNotPersisted")
	})
}

// TestAdversarialContextCancelledDuringRetry verifies a cancelled context skips the second attempt.
func TestAdversarialContextCancelledDuringRetry(t *testing.T) {
	t.Run("cancelled while the first persist runs", func(t *testing.T) {
		bus := &recordingBus{}
		f := newAdvFixture(t, nil, bus, true)
		ctx, cancel := context.WithCancel(advContext())
		defer cancel()
		f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
		f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, interface{}, string, string, *domainDTE.TransmissionEvidence) error {
				cancel()
				return context.Canceled
			}).Times(1)

		start := time.Now()
		_, _, err := f.useCase.Create(ctx, struct{}{})

		requireServiceError(t, err, "StampedDocumentNotPersisted")
		assert.Less(t, time.Since(start), 150*time.Millisecond, "must not wait the retry delay")
		assert.Len(t, bus.events, 1, "the failure must still be reported to the administrator")
	})

	t.Run("cancelled while confirming the reservation", func(t *testing.T) {
		f := newAdvFixture(t, nil, &recordingBus{}, true)
		ctx, cancel := context.WithCancel(advContext())
		defer cancel()
		f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).
			DoAndReturn(func(context.Context, string, string, uint) error {
				cancel()
				return context.Canceled
			}).Times(1)
		f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		_, _, err := f.useCase.Create(ctx, struct{}{})
		assert.NoError(t, err, "a stamped document is still persisted when the confirmation fails")
	})
}

// TestAdversarialRetryDelayIsApplied verifies the second attempt waits before running.
func TestAdversarialRetryDelayIsApplied(t *testing.T) {
	f := newAdvFixture(t, nil, &recordingBus{}, true)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
	var attempts []time.Time
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, interface{}, string, string, *domainDTE.TransmissionEvidence) error {
			attempts = append(attempts, time.Now())
			if len(attempts) == 1 {
				return errors.New("lock wait timeout")
			}
			return nil
		}).Times(2)

	_, _, err := f.useCase.Create(advContext(), struct{}{})

	require.NoError(t, err)
	require.Len(t, attempts, 2)
	assert.GreaterOrEqual(t, attempts[1].Sub(attempts[0]), 190*time.Millisecond)
}

// TestAdversarialConfirmReservationFailsTwice verifies persistence still runs and the client gets a success.
func TestAdversarialConfirmReservationFailsTwice(t *testing.T) {
	bus := &recordingBus{}
	f := newAdvFixture(t, nil, bus, true)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(errors.New("confirm failed")).Times(2)
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, options, err := f.useCase.Create(advContext(), struct{}{})

	require.NoError(t, err)
	assert.Equal(t, testStamp, *options.ReceptionStamp)
	assert.Empty(t, bus.events)
}

// TestAdversarialConfirmFailsAndPersistFails verifies both failures are reported as a single stamped-not-persisted error.
func TestAdversarialConfirmFailsAndPersistFails(t *testing.T) {
	bus := &recordingBus{}
	f := newAdvFixture(t, nil, bus, true)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(errors.New("confirm failed")).Times(2)
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db down")).Times(2)

	_, _, err := f.useCase.Create(advContext(), struct{}{})

	requireServiceError(t, err, "StampedDocumentNotPersisted")
	require.Len(t, bus.events, 1)
}

// TestAdversarialAdditionalOperationsErrors verifies wrapping rules for typed and untyped follow-up errors.
func TestAdversarialAdditionalOperationsErrors(t *testing.T) {
	t.Run("untyped is wrapped and not contingency", func(t *testing.T) {
		ops := func(context.Context, interface{}, uint, interface{}) error { return errors.New("gorm: deadlock") }
		f := newAdvFixture(t, ops, &recordingBus{}, true)
		f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
		f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		_, _, err := f.useCase.Create(advContext(), struct{}{})
		svcErr := requireServiceError(t, err, "AdditionalOperationsFailed")
		assert.Contains(t, svcErr.Message, testGenerationCode)
	})

	t.Run("typed validation error is returned untouched", func(t *testing.T) {
		typed := dte_errors.NewValidationError("InvalidAmount", "x")
		ops := func(context.Context, interface{}, uint, interface{}) error { return typed }
		f := newAdvFixture(t, ops, &recordingBus{}, true)
		f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
		f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		_, _, err := f.useCase.Create(advContext(), struct{}{})
		assert.ErrorIs(t, err, typed)
	})

	t.Run("typed service error is returned untouched", func(t *testing.T) {
		typed := shared_error.NewGeneralServiceError("Ops", "Run", "balance", nil)
		ops := func(context.Context, interface{}, uint, interface{}) error { return typed }
		f := newAdvFixture(t, ops, &recordingBus{}, true)
		f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
		f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		_, _, err := f.useCase.Create(advContext(), struct{}{})
		assert.ErrorIs(t, err, typed)
	})
}

// TestAdversarialStampedEventPayload verifies the event never exposes the signed document and carries every recovery field.
func TestAdversarialStampedEventPayload(t *testing.T) {
	bus := &recordingBus{}
	f := newAdvFixture(t, nil, bus, true)
	f.seqManager.EXPECT().ConfirmReservation(gomock.Any(), testControlNumber, testGenerationCode, uint(1)).Return(nil)
	f.dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db down")).Times(2)

	_, _, _ = f.useCase.Create(advContext(), struct{}{})

	require.Len(t, bus.events, 1)
	evt := bus.events[0].(event.StampedDocumentNotPersistedEvent)
	payload := evt.Payload()
	for _, key := range []string{"branch_id", "nit", "control_number", "generation_code", "dte_type", "reception_stamp", "processing_date", "last_error"} {
		assert.Contains(t, payload, key)
	}
	for _, value := range payload {
		assert.NotContains(t, strings.ToLower(strings.TrimSpace(toString(value))), "signed.jws.document")
	}
	assert.Equal(t, "dte:"+testGenerationCode, evt.AggregateID())
	assert.False(t, evt.OccurredAt().IsZero())
	assert.Equal(t, "NIT-1", evt.NIT)
}

func toString(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
