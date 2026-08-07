package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chainedpixel/ordo-factus/config/drivers"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/jobs"
)

func newCleanerMockDbConnection(t *testing.T) (*drivers.DbConnection, sqlmock.Sqlmock) {
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

func TestReservationCleanerJob_Execute(t *testing.T) {
	test.TestMain(t)

	type mockSetup struct {
		repo func(*mocks.MockReservedSequenceRepositoryPort)
	}

	cases := []struct {
		name  string
		setup mockSetup
	}{
		{
			name: "success – no expired reservations found",
			setup: mockSetup{
				repo: func(m *mocks.MockReservedSequenceRepositoryPort) {
					m.EXPECT().GetExpiredNonContingencyReservations(gomock.Any()).Return([]dte_documents.ReservedSequence{}, nil)
				},
			},
		},
		{
			name: "success – releases expired reservations",
			setup: mockSetup{
				repo: func(m *mocks.MockReservedSequenceRepositoryPort) {
					expired := []dte_documents.ReservedSequence{
						{
							ID:             1,
							BranchID:       10,
							DTEType:        "01",
							SequenceNumber: 100,
							Year:           2026,
							Status:         dte_documents.ReservationStatusReserved,
						},
						{
							ID:             2,
							BranchID:       10,
							DTEType:        "03",
							SequenceNumber: 200,
							Year:           2026,
							Status:         dte_documents.ReservationStatusReserved,
						},
					}
					m.EXPECT().GetExpiredNonContingencyReservations(gomock.Any()).Return(expired, nil)
					m.EXPECT().UpdateStatus(gomock.Any(), uint(10), "01", uint(100), 2026, dte_documents.ReservationStatusReleased, gomock.Any()).Return(nil)
					m.EXPECT().UpdateStatus(gomock.Any(), uint(10), "03", uint(200), 2026, dte_documents.ReservationStatusReleased, gomock.Any()).Return(nil)
				},
			},
		},
		{
			name: "error – query for expired reservations fails",
			setup: mockSetup{
				repo: func(m *mocks.MockReservedSequenceRepositoryPort) {
					m.EXPECT().GetExpiredNonContingencyReservations(gomock.Any()).Return(nil, errors.New("database error"))
				},
			},
		},
		{
			name: "partial error – one update fails, continues with next",
			setup: mockSetup{
				repo: func(m *mocks.MockReservedSequenceRepositoryPort) {
					expired := []dte_documents.ReservedSequence{
						{
							ID:             1,
							BranchID:       10,
							DTEType:        "01",
							SequenceNumber: 100,
							Year:           2026,
						},
						{
							ID:             2,
							BranchID:       20,
							DTEType:        "03",
							SequenceNumber: 200,
							Year:           2026,
						},
					}
					m.EXPECT().GetExpiredNonContingencyReservations(gomock.Any()).Return(expired, nil)
					m.EXPECT().UpdateStatus(gomock.Any(), uint(10), "01", uint(100), 2026, dte_documents.ReservationStatusReleased, gomock.Any()).Return(errors.New("update failed"))
					m.EXPECT().UpdateStatus(gomock.Any(), uint(20), "03", uint(200), 2026, dte_documents.ReservationStatusReleased, gomock.Any()).Return(nil)
				},
			},
		},
		{
			name: "error – context deadline exceeded",
			setup: mockSetup{
				repo: func(m *mocks.MockReservedSequenceRepositoryPort) {
					m.EXPECT().GetExpiredNonContingencyReservations(gomock.Any()).Return(nil, context.DeadlineExceeded)
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockReservedSequenceRepositoryPort(ctrl)
			tc.setup.repo(mockRepo)

			conn, _ := newCleanerMockDbConnection(t)

			job := jobs.NewReservationCleanerJob(mockRepo, conn)
			job.Execute()
		})
	}
}

func TestReservationCleanerJob_ConcurrentExecution(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockReservedSequenceRepositoryPort(ctrl)
	conn, _ := newCleanerMockDbConnection(t)

	blockCh := make(chan struct{})
	mockRepo.EXPECT().GetExpiredNonContingencyReservations(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]dte_documents.ReservedSequence, error) {
		<-blockCh
		return []dte_documents.ReservedSequence{}, nil
	})

	job := jobs.NewReservationCleanerJob(mockRepo, conn)

	go job.Execute()

	time.Sleep(50 * time.Millisecond)
	assert.True(t, job.IsRunning.Load(), "job should be marked as running")

	job.Execute()

	close(blockCh)
	time.Sleep(50 * time.Millisecond)
	assert.False(t, job.IsRunning.Load(), "job should be marked as not running after completion")
}
