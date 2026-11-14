package repositories

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

const pendingSignedJWS = "eyJhbGciOiJSUzUxMiJ9.eyJpZGVudGlmaWNhY2lvbiI6e319.c2lnbmF0dXJl"

// expectPendingContingencyQueries registers the pending query and its preloads for two pending documents whose
// signed document values are given by the caller.
func expectPendingContingencyQueries(mock sqlmock.Sqlmock, firstSigned, secondSigned interface{}) {
	mock.MatchExpectationsInOrder(false)

	mock.ExpectQuery("FROM `contingency_documents`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "branch_id", "type", "reason"}).
			AddRow("c-1", "doc-1", 1, 1, "reason one").
			AddRow("c-2", "doc-2", 1, 1, "reason two"))

	mock.ExpectQuery("FROM `dte_details`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "dte_type", "control_number", "transmission", "status", "json_data", "signed_document"}).
			AddRow("doc-1", "01", "DTE-01-00000001-000000000000001", "contingency", "PENDING", "{}", firstSigned).
			AddRow("doc-2", "03", "DTE-03-00000001-000000000000002", "contingency", "PENDING", "{}", secondSigned))

	mock.ExpectQuery("FROM `branch_offices`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "api_key", "api_secret", "establishment_type", "is_active"}).
			AddRow(1, 7, "key", "secret", "01", true))

	mock.ExpectQuery("FROM `users`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "nit"}).AddRow(7, "0614-010101-000-0"))

	mock.ExpectQuery("FROM `addresses`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id"}))
}

// TestContingencyRepository_GetPending_MapsSignedDocument verifies that the stored JWS of each pending contingency
// document reaches the domain model and that a NULL signed document becomes an empty string.
func TestContingencyRepository_GetPending_MapsSignedDocument(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)
	expectPendingContingencyQueries(mock, pendingSignedJWS, nil)

	repo := repositories.NewContingencyRepository(gormDB)
	docs, err := repo.GetPending(dteRepoCtx(), 10)
	require.NoError(t, err)
	require.Len(t, docs, 2)

	byID := map[string]string{}
	for _, doc := range docs {
		require.NotNil(t, doc.Document)
		byID[doc.Document.ID] = doc.Document.SignedDocument
	}

	assert.Equal(t, pendingSignedJWS, byID["doc-1"], "the stored JWS is mapped unchanged")
	assert.Equal(t, "", byID["doc-2"], "a NULL signed_document is mapped to an empty string")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestContingencyRepository_GetPending_EmptySignedDocumentStaysEmpty verifies that an empty stored value is not
// confused with a signature.
func TestContingencyRepository_GetPending_EmptySignedDocumentStaysEmpty(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newDTERepoMockDB(t)
	expectPendingContingencyQueries(mock, "", "")

	repo := repositories.NewContingencyRepository(gormDB)
	docs, err := repo.GetPending(dteRepoCtx(), 10)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	for _, doc := range docs {
		assert.Empty(t, doc.Document.SignedDocument)
	}
}
