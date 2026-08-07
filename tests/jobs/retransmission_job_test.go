package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chainedpixel/ordo-factus/config/drivers"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/jobs"
)

func newMockDbConnection(t *testing.T) (*drivers.DbConnection, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	mock.ExpectPing()

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	return &drivers.DbConnection{Db: gormDB}, mock
}

func TestRetransmissionJob_Execute(t *testing.T) {
	test.TestMain(t)

	type mockSetup struct {
		contingency func(*mocks.MockContingencyManager)
	}

	cases := []struct {
		name  string
		setup mockSetup
	}{
		{
			name: "success – retransmits pending documents",
			setup: mockSetup{
				contingency: func(m *mocks.MockContingencyManager) {
					m.EXPECT().RetransmitPendingDocuments(gomock.Any()).Return(nil)
				},
			},
		},
		{
			name: "error – retransmission fails",
			setup: mockSetup{
				contingency: func(m *mocks.MockContingencyManager) {
					m.EXPECT().RetransmitPendingDocuments(gomock.Any()).Return(errors.New("hacienda unavailable"))
				},
			},
		},
		{
			name: "error – context deadline exceeded",
			setup: mockSetup{
				contingency: func(m *mocks.MockContingencyManager) {
					m.EXPECT().RetransmitPendingDocuments(gomock.Any()).Return(context.DeadlineExceeded)
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockContingency := mocks.NewMockContingencyManager(ctrl)
			tc.setup.contingency(mockContingency)

			conn, _ := newMockDbConnection(t)

			job := jobs.NewRetransmissionJob(mockContingency, conn)
			job.Execute()
		})
	}
}

func TestRetransmissionJob_ConcurrentExecution(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockContingency := mocks.NewMockContingencyManager(ctrl)
	conn, _ := newMockDbConnection(t)

	blockCh := make(chan struct{})
	mockContingency.EXPECT().RetransmitPendingDocuments(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		<-blockCh
		return nil
	})

	job := jobs.NewRetransmissionJob(mockContingency, conn)

	go job.Execute()

	time.Sleep(50 * time.Millisecond)
	assert.True(t, job.IsRunning.Load(), "job should be marked as running")

	job.Execute()

	close(blockCh)
	time.Sleep(50 * time.Millisecond)
	assert.False(t, job.IsRunning.Load(), "job should be marked as not running after completion")
}
