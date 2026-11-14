package repositories

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

var listColumns = []string{"id", "json_data", "status", "transmission", "hacienda_response", "created_at"}

// TestDTERepository_GetPagedDocuments_MapsObservations verifies the stored Hacienda response is selected and mapped.
func TestDTERepository_GetPagedDocuments_MapsObservations(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)
	now := time.Now()

	mock.ExpectQuery("dte_details.hacienda_response").
		WillReturnRows(sqlmock.NewRows(listColumns).
			AddRow("d-1", `{"a":1}`, "RECEIVED", "normal", `{"estado":"PROCESADO","observaciones":["first","segunda observación"]}`, now).
			AddRow("d-2", `{"a":2}`, "RECEIVED", "normal", nil, now).
			AddRow("d-3", `{"a":3}`, "REJECTED", "contingency", `{"estado":"RECHAZADO","observaciones":["",""]}`, now).
			AddRow("d-4", `{"a":4}`, "RECEIVED", "normal", "{corrupt", now).
			AddRow("d-5", `{"a":5}`, "RECEIVED", "normal", "", now))

	repo := repositories.NewDTERepository(gormDB)
	result, err := repo.GetPagedDocuments(dteRepoCtx(), &dte.DTEFilters{BranchID: 1, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, result, 5)

	assert.Equal(t, []string{"first", "segunda observación"}, result[0].Observations)
	for i := 1; i < 5; i++ {
		assert.Equal(t, []string{}, result[i].Observations, "document %d", i+1)
	}
	assert.Equal(t, "REJECTED", result[2].Status)
	assert.JSONEq(t, `{"a":1}`, string(result[0].Document))

	for _, item := range result {
		raw, marshalErr := json.Marshal(item)
		require.NoError(t, marshalErr)
		var decoded map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &decoded))
		require.Contains(t, decoded, "observations")
		assert.NotEqual(t, "null", string(decoded["observations"]))
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDTERepository_GetPagedDocuments_EmptyResult verifies an empty list is returned without observations.
func TestDTERepository_GetPagedDocuments_EmptyResult(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)
	mock.ExpectQuery("dte_details.hacienda_response").WillReturnRows(sqlmock.NewRows(listColumns))

	result, err := repositories.NewDTERepository(gormDB).GetPagedDocuments(dteRepoCtx(), &dte.DTEFilters{BranchID: 1})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func expectBatchItemQueries(mock sqlmock.Sqlmock, contingencyID, documentID string) {
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `contingency_documents`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("FROM `contingency_documents`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "branch_id", "type", "reason"}).
			AddRow(contingencyID, documentID, 1, 1, "reason"))
}

// TestContingencyRepository_UpdateBatch_ProcessedStoresStampDateAndResponse verifies a stamped document stores everything.
func TestContingencyRepository_UpdateBatch_ProcessedStoresStampDateAndResponse(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)
	response := `{"estado":"PROCESADO","observaciones":["warning"]}`

	expectBatchItemQueries(mock, "c-1", "doc-1")
	mock.ExpectExec("UPDATE `dte_details` SET `hacienda_response`=\\?,`processing_date`=\\?,`reception_stamp`=\\?,`status`=\\? WHERE id = \\?").
		WithArgs(response, "14/11/2026 10:00:00", "STAMP", constants.DocumentReceived, "doc-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := repositories.NewContingencyRepository(gormDB)
	err := repo.UpdateBatch(dteRepoCtx(), []string{"c-1"}, []string{"RECIBIDO"}, map[string]dte.ReceptionResult{
		"c-1": {ReceptionStamp: "STAMP", ProcessingDate: "14/11/2026 10:00:00", HaciendaResponse: response},
	}, "batch-1", "mh-1", constants.DocumentReceived)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestContingencyRepository_UpdateBatch_RejectedStoresResponseWithoutStamp verifies a rejected document keeps its
// response and status but never gets a stamp or processing date.
func TestContingencyRepository_UpdateBatch_RejectedStoresResponseWithoutStamp(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)
	response := `{"estado":"RECHAZADO","observaciones":["error"]}`

	expectBatchItemQueries(mock, "c-1", "doc-1")
	mock.ExpectExec("UPDATE `dte_details` SET `hacienda_response`=\\?,`status`=\\? WHERE id = \\?").
		WithArgs(response, constants.DocumentRejected, "doc-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := repositories.NewContingencyRepository(gormDB)
	err := repo.UpdateBatch(dteRepoCtx(), []string{"c-1"}, []string{"RECHAZADO"}, map[string]dte.ReceptionResult{
		"c-1": {HaciendaResponse: response},
	}, "batch-1", "mh-1", constants.DocumentRejected)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestContingencyRepository_UpdateBatch_EvidenceVariants covers partial results.
func TestContingencyRepository_UpdateBatch_EvidenceVariants(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name     string
		results  map[string]dte.ReceptionResult
		updateRE string
		args     []driver.Value
	}{
		{
			name:     "no result for the document updates only the status",
			results:  map[string]dte.ReceptionResult{},
			updateRE: "UPDATE `dte_details` SET `status`=\\? WHERE id = \\?",
			args:     []driver.Value{constants.DocumentRejected, "doc-1"},
		},
		{
			name:     "nil results map updates only the status",
			results:  nil,
			updateRE: "UPDATE `dte_details` SET `status`=\\? WHERE id = \\?",
			args:     []driver.Value{constants.DocumentRejected, "doc-1"},
		},
		{
			name:     "empty response is not written",
			results:  map[string]dte.ReceptionResult{"c-1": {ReceptionStamp: "STAMP"}},
			updateRE: "UPDATE `dte_details` SET `reception_stamp`=\\?,`status`=\\? WHERE id = \\?",
			args:     []driver.Value{"STAMP", constants.DocumentRejected, "doc-1"},
		},
		{
			name:     "processing date without stamp is not written",
			results:  map[string]dte.ReceptionResult{"c-1": {ProcessingDate: "14/11/2026 10:00:00", HaciendaResponse: "{}"}},
			updateRE: "UPDATE `dte_details` SET `hacienda_response`=\\?,`status`=\\? WHERE id = \\?",
			args:     []driver.Value{"{}", constants.DocumentRejected, "doc-1"},
		},
		{
			name:     "stamp without processing date writes the stamp only",
			results:  map[string]dte.ReceptionResult{"c-1": {ReceptionStamp: "STAMP", HaciendaResponse: "{}"}},
			updateRE: "UPDATE `dte_details` SET `hacienda_response`=\\?,`reception_stamp`=\\?,`status`=\\? WHERE id = \\?",
			args:     []driver.Value{"{}", "STAMP", constants.DocumentRejected, "doc-1"},
		},
		{
			name:     "result for another document is ignored",
			results:  map[string]dte.ReceptionResult{"other": {ReceptionStamp: "STAMP", HaciendaResponse: "{}"}},
			updateRE: "UPDATE `dte_details` SET `status`=\\? WHERE id = \\?",
			args:     []driver.Value{constants.DocumentRejected, "doc-1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newDTERepoMockDB(t)
			expectBatchItemQueries(mock, "c-1", "doc-1")
			mock.ExpectExec(tc.updateRE).WithArgs(tc.args...).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			err := repositories.NewContingencyRepository(gormDB).
				UpdateBatch(dteRepoCtx(), []string{"c-1"}, nil, tc.results, "batch-1", "mh-1", constants.DocumentRejected)

			require.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestContingencyRepository_UpdateBatch_ResponsesAreStoredPerDocument verifies each document gets its own response.
func TestContingencyRepository_UpdateBatch_ResponsesAreStoredPerDocument(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)

	mock.ExpectBegin()
	for _, item := range []struct{ contingencyID, documentID, response string }{
		{"c-1", "doc-1", `{"codigoGeneracion":"one"}`},
		{"c-2", "doc-2", `{"codigoGeneracion":"two"}`},
	} {
		mock.ExpectExec("UPDATE `contingency_documents`").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("FROM `contingency_documents`").
			WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "branch_id", "type", "reason"}).
				AddRow(item.contingencyID, item.documentID, 1, 1, "reason"))
		mock.ExpectExec("UPDATE `dte_details` SET `hacienda_response`=\\?,`status`=\\? WHERE id = \\?").
			WithArgs(item.response, constants.DocumentRejected, item.documentID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	err := repositories.NewContingencyRepository(gormDB).UpdateBatch(dteRepoCtx(), []string{"c-1", "c-2"}, []string{"a", "b"},
		map[string]dte.ReceptionResult{
			"c-1": {HaciendaResponse: `{"codigoGeneracion":"one"}`},
			"c-2": {HaciendaResponse: `{"codigoGeneracion":"two"}`},
		}, "batch-1", "mh-1", constants.DocumentRejected)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
