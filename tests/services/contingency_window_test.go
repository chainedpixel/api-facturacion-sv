package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	coreDte "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/event"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/contingency"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

func TestIsContingencyWindowExpired(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	within := now.Add(-71 * time.Hour)
	over := now.Add(-73 * time.Hour)

	assert.False(t, contingency.IsContingencyWindowExpired(nil, now), "no event stamp never expires")
	assert.False(t, contingency.IsContingencyWindowExpired(&within, now))
	assert.True(t, contingency.IsContingencyWindowExpired(&over, now))
}

// TestContingencyService_ExpiresDocumentsAfter72Hours checks that an expired document is marked and not retransmitted.
func TestContingencyService_ExpiresDocumentsAfter72Hours(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockEvent := mocks.NewMockContingencyEventSender(ctrl)

	stampedAt := time.Now().Add(-73 * time.Hour)
	expired := buildPendingDoc("0614-010101-000-0", "01")
	expired.EventStampedAt = &stampedAt

	mockRepo.EXPECT().GetPending(gomock.Any(), gomock.Any()).
		Return([]coreDte.ContingencyDocument{expired}, nil)
	mockRepo.EXPECT().
		UpdateBatch(gomock.Any(), []string{expired.ID}, gomock.Any(), gomock.Nil(), "expired", "", constants.DocumentExpired).
		Return(nil)
	mockEvent.EXPECT().PrepareAndSendContingencyEvent(gomock.Any(), gomock.Any()).Times(0)

	svc := contingency.NewContingencyManager(
		nil, mocks.NewMockDTEManager(ctrl), mockRepo,
		nil, nil, nil, nil, nil,
		mockEvent, mocks.NewMockSequentialNumberManager(ctrl), nil, nil,
	)

	assert.NoError(t, svc.RetransmitPendingDocuments(context.Background()))
}

// recordingBus stores published events to assert administrator alerts.
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

func (b *recordingBus) jobNames() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var names []string
	for _, evt := range b.events {
		if failed, ok := evt.(event.RetransmissionJobFailedEvent); ok {
			names = append(names, failed.JobName)
		}
	}
	return names
}

// TestContingencyService_AdminAlerts checks the expired documents and long contingency alerts.
func TestContingencyService_AdminAlerts(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockEvent := mocks.NewMockContingencyEventSender(ctrl)

	stampedAt := time.Now().Add(-73 * time.Hour)
	expired := buildPendingDoc("0614-010101-000-0", "01")
	expired.EventStampedAt = &stampedAt

	longRunning := buildPendingDoc("0614-010101-000-0", "03")
	longRunning.ID = "contingency-2"
	longRunning.CreatedAt = time.Now().Add(-4 * 24 * time.Hour)

	mockRepo.EXPECT().GetPending(gomock.Any(), gomock.Any()).
		Return([]coreDte.ContingencyDocument{expired, longRunning}, nil)
	mockRepo.EXPECT().UpdateBatch(gomock.Any(), []string{expired.ID}, gomock.Any(), gomock.Nil(), "expired", "", constants.DocumentExpired).
		Return(nil)
	mockEvent.EXPECT().PrepareAndSendContingencyEvent(gomock.Any(), gomock.Len(1)).Return(errors.New("stop"))

	svc := contingency.NewContingencyManager(
		nil, mocks.NewMockDTEManager(ctrl), mockRepo,
		nil, nil, nil, nil, nil,
		mockEvent, mocks.NewMockSequentialNumberManager(ctrl), nil, nil,
	)
	bus := &recordingBus{}
	svc.(*contingency.ContingencyService).SetEventBus(bus)

	assert.NoError(t, svc.RetransmitPendingDocuments(context.Background()))
	assert.ElementsMatch(t, []string{"contingency_72h_expired", "contingency_technical_report"}, bus.jobNames())
}
