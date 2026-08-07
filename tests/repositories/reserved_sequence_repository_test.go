package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func newReservedSequenceMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	return gormDB, mock
}

// TestReservedSequenceRepository_GetOldestReleasedNumber_FiltersHaciendaRejected guards
// the regression where a control number rejected by Hacienda (status=RELEASED with a
// non-empty hacienda_code) was reused on the next emission, retransmitted, rejected
// again and locked the branch on the same number forever.
//
// After the fix, GetOldestReleasedNumber must only return reservations whose
// hacienda_code is NULL or empty (released by TTL expiry or local errors before
// reaching Hacienda).
func TestReservedSequenceRepository_GetOldestReleasedNumber_FiltersHaciendaRejected(t *testing.T) {
	test.TestMain(t)

	t.Run("query must include hacienda_code IS NULL OR empty filter", func(t *testing.T) {
		gormDB, mock := newReservedSequenceMockDB(t)
		repo := repositories.NewReservedSequenceRepository(gormDB)

		// Match the SQL emitted by GORM for the query, ensuring the filter is present.
		// We don't pin the exact SQL, only assert that the hacienda_code clause is there.
		mock.ExpectQuery("hacienda_code IS NULL OR hacienda_code = ''").
			WithArgs(uint(1), "01", 2026, dte_documents.ReservationStatusReleased, 1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "branch_id", "dte_type", "sequence_number", "year",
				"status", "document_id", "reserved_at", "confirmed_at",
				"released_at", "expires_at", "rejection_reason", "hacienda_code",
				"is_contingency", "contingency_document_id",
			}).AddRow(
				42, 1, "01", 7959, 2026,
				dte_documents.ReservationStatusReleased, nil, time.Now(), nil,
				time.Now(), nil, nil, nil,
				false, nil,
			))

		result, err := repo.GetOldestReleasedNumber(context.Background(), 1, "01", 2026)
		assert.NoError(t, err)
		assert.NotNil(t, result, "released reservation without hacienda_code must be reusable")
		assert.Equal(t, uint(7959), result.SequenceNumber)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns nil when only MH-rejected released numbers exist", func(t *testing.T) {
		gormDB, mock := newReservedSequenceMockDB(t)
		repo := repositories.NewReservedSequenceRepository(gormDB)

		// Simulates the production scenario: every RELEASED row in the DB has a
		// hacienda_code (e.g. "004"), so the filtered query returns no rows.
		mock.ExpectQuery("hacienda_code IS NULL OR hacienda_code = ''").
			WithArgs(uint(1), "01", 2026, dte_documents.ReservationStatusReleased, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		result, err := repo.GetOldestReleasedNumber(context.Background(), 1, "01", 2026)
		assert.NoError(t, err, "ErrRecordNotFound must be surfaced as (nil, nil)")
		assert.Nil(t, result, "no reusable number should be returned when only MH-rejected ones exist")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
