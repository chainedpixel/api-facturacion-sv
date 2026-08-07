package repositories

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func newDTERepoMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func dteRepoCtx() context.Context {
	return context.WithValue(context.Background(), "claims", &authModels.AuthClaims{
		ClientID: 1,
		BranchID: 1,
		NIT:      "0614-010101-000-0",
	})
}

// TestDTERepository_Create_RejectsEmptyIdentification guards the regression where
// a DTE with an empty GenerationCode, ControlNumber or DTEType was persisted into
// dte_documents/dte_details with id="", leaving rows that had to be cleaned up by
// hand. After the fix, Create must abort with the MissingDTEIdentification i18n
// code and never issue an INSERT.
func TestDTERepository_Create_RejectsEmptyIdentification(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name     string
		document map[string]interface{}
	}{
		{
			name: "empty GenerationCode",
			document: map[string]interface{}{
				"identificacion": map[string]interface{}{
					"tipoDte":          "01",
					"numeroControl":    "DTE-01-00000001-000000000000001",
					"codigoGeneracion": "",
				},
			},
		},
		{
			name: "empty ControlNumber",
			document: map[string]interface{}{
				"identificacion": map[string]interface{}{
					"tipoDte":          "01",
					"numeroControl":    "",
					"codigoGeneracion": "AB123456-1234-1234-1234-AB1234567890",
				},
			},
		},
		{
			name: "empty DTEType",
			document: map[string]interface{}{
				"identificacion": map[string]interface{}{
					"tipoDte":          "",
					"numeroControl":    "DTE-01-00000001-000000000000001",
					"codigoGeneracion": "AB123456-1234-1234-1234-AB1234567890",
				},
			},
		},
		{
			name: "all identification fields missing",
			document: map[string]interface{}{
				"emisor": map[string]interface{}{"nit": "0614-010101-000-0"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newDTERepoMockDB(t)
			repo := repositories.NewDTERepository(gormDB)

			// We deliberately set zero expectations on the mock: Create must not
			// reach the database when identification fields are empty.

			err := repo.Create(dteRepoCtx(), tc.document, constants.TransmissionNormal, constants.DocumentReceived, nil)

			assert.Error(t, err)
			test.AssertErrorCode(t, err, "MissingDTEIdentification")
			assert.NoError(t, mock.ExpectationsWereMet(), "no SQL should have been executed")
		})
	}
}
