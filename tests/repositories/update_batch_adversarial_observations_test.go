package repositories

import (
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

// TestAdversarialObservations_UpdateBatchStoresHostileResponsesVerbatim checks hostile text is bound as a parameter
// and stored exactly as received.
func TestAdversarialObservations_UpdateBatchStoresHostileResponsesVerbatim(t *testing.T) {
	test.TestMain(t)

	responses := map[string]string{
		"sql injection":      `{"observaciones":["'; DROP TABLE dte_details; --"]}`,
		"html":               `{"observaciones":["<script>alert(1)</script>"]}`,
		"control characters": "{\"observaciones\":[\"a\\u0000b\"]}\x00\x01",
		"not json":           "plain text \xff\xfe",
		"one megabyte":       `{"observaciones":["` + strings.Repeat("x", 1<<20) + `"]}`,
		"whitespace only":    "  ",
	}
	for name, response := range responses {
		t.Run(name, func(t *testing.T) {
			gormDB, mock := newDTERepoMockDB(t)
			expectBatchItemQueries(mock, "c-1", "doc-1")
			mock.ExpectExec("UPDATE `dte_details` SET `hacienda_response`=\\?,`status`=\\? WHERE id = \\?").
				WithArgs(response, constants.DocumentRejected, "doc-1").
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			err := repositories.NewContingencyRepository(gormDB).UpdateBatch(dteRepoCtx(), []string{"c-1"}, []string{"obs"},
				map[string]dte.ReceptionResult{"c-1": {HaciendaResponse: response}}, "batch-1", "mh-1", constants.DocumentRejected)

			require.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestAdversarialObservations_UpdateBatchEmptyResponsesNeverWriteTheColumn checks that results without a Hacienda
// response never write hacienda_response, for each document independently.
func TestAdversarialObservations_UpdateBatchEmptyResponsesNeverWriteTheColumn(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)
	mock.ExpectBegin()
	items := []struct {
		contingencyID, documentID, updateRE string
		args                                []interface{}
	}{
		{"c-1", "doc-1", "UPDATE `dte_details` SET `status`=\\? WHERE id = \\?", []interface{}{constants.DocumentReceived, "doc-1"}},
		{"c-2", "doc-2", "UPDATE `dte_details` SET `hacienda_response`=\\?,`status`=\\? WHERE id = \\?", []interface{}{`{"a":1}`, constants.DocumentReceived, "doc-2"}},
		{"c-3", "doc-3", "UPDATE `dte_details` SET `status`=\\? WHERE id = \\?", []interface{}{constants.DocumentReceived, "doc-3"}},
		{"c-4", "doc-4", "UPDATE `dte_details` SET `reception_stamp`=\\?,`status`=\\? WHERE id = \\?", []interface{}{"STAMP", constants.DocumentReceived, "doc-4"}},
	}
	for _, item := range items {
		mock.ExpectExec("UPDATE `contingency_documents`").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("FROM `contingency_documents`").
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "branch_id", "type", "reason"}).
				AddRow(item.contingencyID, item.documentID, 1, 1, "reason"))
		args := make([]driver.Value, len(item.args))
		for i, arg := range item.args {
			args[i] = arg
		}
		mock.ExpectExec(item.updateRE).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	err := repositories.NewContingencyRepository(gormDB).UpdateBatch(dteRepoCtx(),
		[]string{"c-1", "c-2", "c-3", "c-4"}, nil,
		map[string]dte.ReceptionResult{
			"c-1": {},
			"c-2": {HaciendaResponse: `{"a":1}`},
			"c-3": {ProcessingDate: "14/11/2026 10:00:00"},
			"c-4": {ReceptionStamp: "STAMP"},
			"c-9": {HaciendaResponse: `{"unknown":true}`},
		}, "batch-1", "mh-1", constants.DocumentReceived)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAdversarialObservations_UpdateBatchDegenerateInputs checks empty ids and mismatched observation lists.
func TestAdversarialObservations_UpdateBatchDegenerateInputs(t *testing.T) {
	test.TestMain(t)

	t.Run("no ids only opens and commits", func(t *testing.T) {
		gormDB, mock := newDTERepoMockDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		err := repositories.NewContingencyRepository(gormDB).UpdateBatch(dteRepoCtx(), nil, nil,
			map[string]dte.ReceptionResult{"c-1": {HaciendaResponse: "{}"}}, "batch-1", "mh-1", constants.DocumentReceived)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("more observations than ids are ignored", func(t *testing.T) {
		gormDB, mock := newDTERepoMockDB(t)
		expectBatchItemQueries(mock, "c-1", "doc-1")
		mock.ExpectExec("UPDATE `dte_details` SET `status`=\\? WHERE id = \\?").
			WithArgs(constants.DocumentReceived, "doc-1").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		err := repositories.NewContingencyRepository(gormDB).UpdateBatch(dteRepoCtx(), []string{"c-1"}, []string{"a", "b", "c"},
			nil, "batch-1", "mh-1", constants.DocumentReceived)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("document update failure rolls back and returns the error", func(t *testing.T) {
		gormDB, mock := newDTERepoMockDB(t)
		expectBatchItemQueries(mock, "c-1", "doc-1")
		mock.ExpectExec("UPDATE `dte_details`").WillReturnError(errors.New("write failed"))
		mock.ExpectRollback()
		var err error
		require.NotPanics(t, func() {
			err = repositories.NewContingencyRepository(gormDB).UpdateBatch(dteRepoCtx(), []string{"c-1"}, nil,
				map[string]dte.ReceptionResult{"c-1": {HaciendaResponse: `{"a":1}`}}, "batch-1", "mh-1", constants.DocumentRejected)
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "write failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
