package repositories

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/ordo-factus/tests"
)

// TestControlNumberRepository_GetNext_LocksCounterRow verifies the counter is read with FOR UPDATE inside the
// transaction that increments it.
func TestControlNumberRepository_GetNext_LocksCounterRow(t *testing.T) {
	test.TestMain(t)
	db, mock := newDTERepoMockDB(t)

	mock.ExpectBegin()
	mock.ExpectQuery("FOR UPDATE").
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "dte_type", "last_number", "year"}).
			AddRow(1, 1, "01", 4, 2026))
	mock.ExpectExec("UPDATE `control_number_sequences`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	next, err := repositories.NewControlNumberRepository(db).GetNext(context.Background(), "01", 1)

	require.NoError(t, err)
	assert.Equal(t, 5, next)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestControlNumberRepository_GetNext_RetriesAfterConcurrentFirstInsert verifies that a failed first insert
// (unique index violation) is retried with a new locked read.
func TestControlNumberRepository_GetNext_RetriesAfterConcurrentFirstInsert(t *testing.T) {
	test.TestMain(t)
	db, mock := newDTERepoMockDB(t)
	columns := []string{"id", "branch_id", "dte_type", "last_number", "year"}

	mock.ExpectBegin()
	mock.ExpectQuery("FOR UPDATE").WillReturnRows(sqlmock.NewRows(columns))
	mock.ExpectExec("INSERT INTO `control_number_sequences`").WillReturnError(assert.AnError)
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectQuery("FOR UPDATE").WillReturnRows(sqlmock.NewRows(columns).AddRow(1, 1, "01", 1, 2026))
	mock.ExpectExec("UPDATE `control_number_sequences`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	next, err := repositories.NewControlNumberRepository(db).GetNext(context.Background(), "01", 1)

	require.NoError(t, err)
	assert.Equal(t, 2, next)
	assert.NoError(t, mock.ExpectationsWereMet())
}
