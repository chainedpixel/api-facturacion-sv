package repositories

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/database/db_models"
)

// TestDTEDocumentAfterCreate_BalanceIsNetOfGlobalDiscount checks that the initial balance of a CCF is its
// taxed total minus the global taxed discount (100.00 with 10.00 of discount leaves 90.00).
func TestDTEDocumentAfterCreate_BalanceIsNetOfGlobalDiscount(t *testing.T) {
	gormDB, mock := newDTERepoMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `dte_balance_control`").
		WithArgs(uint(1), "CCF-1", 90.0, 0.0, 0.0, 90.0, 0.0, 0.0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	doc := &db_models.DTEDocument{
		DocumentID: "CCF-1",
		BranchID:   1,
		Document: &db_models.DTEDetails{
			DTEType:  constants.CCFElectronico,
			JSONData: `{"resumen":{"totalGravada":100.00,"totalExenta":0,"totalNoSuj":0,"descuGravada":10.00,"porcentajeDescuento":10}}`,
		},
	}

	require.NoError(t, gormDB.Transaction(func(tx *gorm.DB) error { return doc.AfterCreate(tx) }))
	assert.NoError(t, mock.ExpectationsWereMet())
}
