package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	domainDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	transmitterModels "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/transmitter/hacienda_error"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

func advObsStatusRejection(status string) *hacienda_error.HaciendaResponseError {
	return hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{
		Status: status, MessageCode: "095", Observations: []string{"x"},
	}, 400)
}

// TestAdversarialObservations_RejectedInvalidationStatusVariants checks only the exact RECHAZADO status stores
// evidence, and the original error is always returned unchanged without invalidating the document.
func TestAdversarialObservations_RejectedInvalidationStatusVariants(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		status string
		stored bool
	}{
		{"RECHAZADO", true},
		{"RECHAZADO ", false},
		{" RECHAZADO", false},
		{"rechazado", false},
		{"Rechazado", false},
		{"", false},
		{"RECHAZADOS", false},
		{"RECHAZADO\n", false},
		{"PROCESADO", false},
		{"RЕCHAZADO", false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.status), func(t *testing.T) {
			transmitErr := &ports.TransmissionError{Err: advObsStatusRejection(tc.status), SignedDocument: rejectedInvalidationJWS}
			f := newRejectedInvalidationFixture(t, transmitErr)

			response, err := f.useCase.InvalidateDocument(advContext(), invalidationRequest())

			require.Error(t, err)
			assert.Nil(t, response)
			assert.Same(t, transmitErr, err)
			assert.Equal(t, 0, f.manager.invalidated)
			if tc.stored {
				require.Len(t, f.registered, 1)
				assert.Equal(t, constants.DocumentRejected, f.registered[0].Status)
			} else {
				assert.Empty(t, f.registered)
			}
		})
	}
}

// TestAdversarialObservations_RejectedInvalidationWrapperShapes checks wrappers with empty evidence still keep the
// rejected attempt and never change the returned error.
func TestAdversarialObservations_RejectedInvalidationWrapperShapes(t *testing.T) {
	test.TestMain(t)

	huge := make([]string, 5000)
	for i := range huge {
		huge[i] = fmt.Sprintf("obs <i>%d</i> \"q\"", i)
	}
	hostileResponse := hacienda_error.NewHaciendaError(&transmitterModels.HaciendaResponse{
		Status: hacienda_error.StatusRejected, MessageCode: "095", Observations: huge,
		DescriptionMessage: strings.Repeat("d", 10000),
	}, 400)
	innerOnlyResponse := &hacienda_error.HaciendaResponseError{
		Status: hacienda_error.StatusRejected, Code: "001", Observations: []string{"only in error"},
	}
	responseOverridesError := &hacienda_error.HaciendaResponseError{
		Status: hacienda_error.StatusRejected, Observations: []string{"from error"},
		Response: &transmitterModels.HaciendaResponse{Status: hacienda_error.StatusRejected, Observations: []string{"from response"}},
	}

	cases := []struct {
		name         string
		err          error
		wantJWS      string
		wantObserved []string
	}{
		{"empty jws", &ports.TransmissionError{Err: advObsStatusRejection(hacienda_error.StatusRejected)}, "", []string{"x"}},
		{"nil response in error", &ports.TransmissionError{Err: innerOnlyResponse, SignedDocument: rejectedInvalidationJWS}, rejectedInvalidationJWS, []string{"only in error"}},
		{"response wins over error fields", &ports.TransmissionError{Err: responseOverridesError, SignedDocument: rejectedInvalidationJWS}, rejectedInvalidationJWS, []string{"from response"}},
		{"nested wrappers outer jws wins", &ports.TransmissionError{SignedDocument: "outer.jws.value", Err: &ports.TransmissionError{
			SignedDocument: "inner.jws.value", Err: advObsStatusRejection(hacienda_error.StatusRejected)}}, "outer.jws.value", []string{"x"}},
		{"wrapped by fmt twice", fmt.Errorf("a: %w", fmt.Errorf("b: %w", &ports.TransmissionError{
			Err: advObsStatusRejection(hacienda_error.StatusRejected), SignedDocument: rejectedInvalidationJWS})), rejectedInvalidationJWS, []string{"x"}},
		{"huge hostile observations", &ports.TransmissionError{Err: hostileResponse, SignedDocument: rejectedInvalidationJWS}, rejectedInvalidationJWS, huge},
		{"empty observations", &ports.TransmissionError{Err: advObsRejectionWithoutObservations(), SignedDocument: rejectedInvalidationJWS}, rejectedInvalidationJWS, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newRejectedInvalidationFixture(t, tc.err)

			response, err := f.useCase.InvalidateDocument(advContext(), invalidationRequest())

			require.Error(t, err)
			assert.Nil(t, response)
			assert.Same(t, tc.err, err)
			assert.Equal(t, 0, f.manager.invalidated)

			require.Len(t, f.registered, 1)
			event := f.registered[0]
			assert.Equal(t, constants.DocumentRejected, event.Status)
			assert.Equal(t, tc.wantJWS, event.SignedDocument)
			assert.Nil(t, event.ReceptionStamp)
			assert.True(t, json.Valid([]byte(event.HaciendaResponse)))
			assert.Equal(t, tc.wantObserved, domainDTE.ObservationsFromHaciendaResponse(event.HaciendaResponse))
		})
	}
}

// TestAdversarialObservations_RejectedInvalidationRegisterFailures checks storage errors and a cancelled context never
// replace the original error.
func TestAdversarialObservations_RejectedInvalidationRegisterFailures(t *testing.T) {
	test.TestMain(t)

	failures := map[string]func(*domainDTE.DTEEvent) error{
		"plain error":    func(*domainDTE.DTEEvent) error { return errors.New("database down") },
		"context error":  func(*domainDTE.DTEEvent) error { return context.Canceled },
		"deadline error": func(*domainDTE.DTEEvent) error { return context.DeadlineExceeded },
		"hacienda error": func(*domainDTE.DTEEvent) error { return advObsStatusRejection(hacienda_error.StatusRejected) },
		"wrapper as error": func(*domainDTE.DTEEvent) error {
			return &ports.TransmissionError{Err: errors.New("nested"), SignedDocument: "x"}
		},
	}
	for name, registerFn := range failures {
		t.Run(name, func(t *testing.T) {
			transmitErr := &ports.TransmissionError{Err: advObsStatusRejection(hacienda_error.StatusRejected), SignedDocument: rejectedInvalidationJWS}
			f := newRejectedInvalidationFixture(t, transmitErr)
			f.registerFn = registerFn

			response, err := f.useCase.InvalidateDocument(advContext(), invalidationRequest())

			require.Error(t, err)
			assert.Nil(t, response)
			assert.Same(t, transmitErr, err)
			assert.Equal(t, 0, f.manager.invalidated)
			assert.Len(t, f.registered, 1)
		})
	}

	t.Run("cancelled request context", func(t *testing.T) {
		transmitErr := &ports.TransmissionError{Err: advObsStatusRejection(hacienda_error.StatusRejected), SignedDocument: rejectedInvalidationJWS}
		f := newRejectedInvalidationFixture(t, transmitErr)
		ctx, cancel := context.WithCancel(advContext())
		cancel()
		f.registerFn = func(*domainDTE.DTEEvent) error { return ctx.Err() }

		response, err := f.useCase.InvalidateDocument(ctx, invalidationRequest())

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Same(t, transmitErr, err)
		assert.Equal(t, 0, f.manager.invalidated)
	})
}

// TestAdversarialObservations_PanicWhileStoringRejectedInvalidationIsContained checks a panic of the storage layer
// while keeping the rejected attempt does not replace the original error returned to the consumer.
func TestAdversarialObservations_PanicWhileStoringRejectedInvalidationIsContained(t *testing.T) {
	test.TestMain(t)

	transmitErr := &ports.TransmissionError{Err: advObsStatusRejection(hacienda_error.StatusRejected), SignedDocument: rejectedInvalidationJWS}
	f := newRejectedInvalidationFixture(t, transmitErr)
	f.registerFn = func(*domainDTE.DTEEvent) error { panic("storage layer panic") }

	var err error
	require.NotPanics(t, func() {
		_, err = f.useCase.InvalidateDocument(advContext(), invalidationRequest())
	})
	assert.Same(t, transmitErr, err)
	assert.Equal(t, 0, f.manager.invalidated)
}

func advObsRejectionWithoutObservations() *hacienda_error.HaciendaResponseError {
	rejection := advObsStatusRejection(hacienda_error.StatusRejected)
	rejection.Observations = nil
	rejection.Response.Observations = nil
	return rejection
}
