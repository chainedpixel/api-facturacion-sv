package repositories

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
)

// TestDTEDocumentAfterCreate_BalanceAdversarial attacks the initial balance with unusual summary values.
func TestDTEDocumentAfterCreate_BalanceAdversarial(t *testing.T) {
	cases := []struct {
		name    string
		dteType string
		json    string
		taxed   float64
		exempt  float64
		notSubj float64
	}{
		{"discount greater than total clamps to zero", constants.CCFElectronico, `{"resumen":{"totalGravada":10,"descuGravada":25}}`, 0, 0, 0},
		{"discount equal to total", constants.CCFElectronico, `{"resumen":{"totalGravada":10,"descuGravada":10}}`, 0, 0, 0},
		{"missing descu fields in old documents", constants.CCFElectronico, `{"resumen":{"totalGravada":100,"totalExenta":5,"totalNoSuj":2}}`, 100, 5, 2},
		{"null descu fields", constants.CCFElectronico, `{"resumen":{"totalGravada":100,"descuGravada":null,"descuExenta":null,"descuNoSuj":null}}`, 100, 0, 0},
		{"exempt only with discount", constants.CCFElectronico, `{"resumen":{"totalGravada":0,"totalExenta":50,"descuExenta":12.5}}`, 0, 37.5, 0},
		{"not subject only with discount", constants.CCFElectronico, `{"resumen":{"totalNoSuj":30,"descuNoSuj":0.01}}`, 0, 0, 29.99},
		{"discounts apply per category", constants.CCFElectronico, `{"resumen":{"totalGravada":100,"totalExenta":40,"totalNoSuj":20,"descuGravada":10,"descuExenta":4,"descuNoSuj":2}}`, 90, 36, 18},
		{"discount in one category does not leak to others", constants.CCFElectronico, `{"resumen":{"totalGravada":100,"totalExenta":40,"descuExenta":40}}`, 100, 0, 0},
		{"large amount minus one cent", constants.CCFElectronico, `{"resumen":{"totalGravada":99999999.99,"descuGravada":0.01}}`, 99999999.98, 0, 0},
		{"fractional discount keeps cents", constants.CCFElectronico, `{"resumen":{"totalGravada":33.33,"descuGravada":3.33}}`, 30, 0, 0},
		{"retention voucher without taxed fields stays at zero", constants.ComprobanteRetencionElectronico, `{"resumen":{"totalSujetoRetencion":100,"totalIVAretenido":1}}`, 0, 0, 0},
		{"retention voucher with taxed fields unchanged without discounts", constants.ComprobanteRetencionElectronico, `{"resumen":{"totalGravada":100,"totalExenta":1,"totalNoSuj":2}}`, 100, 1, 2},
		{"empty resumen", constants.CCFElectronico, `{"resumen":{}}`, 0, 0, 0},
		{"no resumen at all", constants.CCFElectronico, `{}`, 0, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newDTERepoMockDB(t)

			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO `dte_balance_control`").
				WithArgs(uint(7), "DOC-1", tc.taxed, tc.exempt, tc.notSubj, tc.taxed, tc.exempt, tc.notSubj).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			doc := &db_models.DTEDocument{
				DocumentID: "DOC-1",
				BranchID:   7,
				Document:   &db_models.DTEDetails{DTEType: tc.dteType, JSONData: tc.json},
			}

			require.NoError(t, gormDB.Transaction(func(tx *gorm.DB) error { return doc.AfterCreate(tx) }))
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestDTEDocumentAfterCreate_BalanceMalformedJSON checks that malformed summary values abort without inserting.
func TestDTEDocumentAfterCreate_BalanceMalformedJSON(t *testing.T) {
	cases := map[string]string{
		"string discount":      `{"resumen":{"totalGravada":100,"descuGravada":"abc"}}`,
		"string total":         `{"resumen":{"totalGravada":"abc"}}`,
		"truncated json":       `{"resumen":{"totalGravada":100`,
		"discount as object":   `{"resumen":{"totalGravada":100,"descuExenta":{}}}`,
		"empty document":       ``,
		"discount as boolean":  `{"resumen":{"totalGravada":100,"descuNoSuj":true}}`,
		"numeric string value": `{"resumen":{"totalGravada":100,"descuGravada":"10"}}`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			gormDB, mock := newDTERepoMockDB(t)
			mock.ExpectBegin()
			mock.ExpectRollback()

			doc := &db_models.DTEDocument{
				DocumentID: "DOC-1",
				BranchID:   1,
				Document:   &db_models.DTEDetails{DTEType: constants.CCFElectronico, JSONData: raw},
			}

			err := gormDB.Transaction(func(tx *gorm.DB) error { return doc.AfterCreate(tx) })
			assert.Error(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestDTEDocumentAfterCreate_NonAdjustableTypesCreateNoBalance checks that other document types never get a balance row.
func TestDTEDocumentAfterCreate_NonAdjustableTypesCreateNoBalance(t *testing.T) {
	for _, dteType := range []string{constants.FacturaElectronica, constants.NotaCreditoElectronica, constants.NotaDebitoElectronica, constants.NotaRemisionElectronica} {
		t.Run(dteType, func(t *testing.T) {
			gormDB, mock := newDTERepoMockDB(t)
			mock.ExpectBegin()
			mock.ExpectCommit()

			doc := &db_models.DTEDocument{
				DocumentID: "DOC-1",
				BranchID:   1,
				Document:   &db_models.DTEDetails{DTEType: dteType, JSONData: `{"resumen":{"totalGravada":100,"descuGravada":10}}`},
			}

			require.NoError(t, gormDB.Transaction(func(tx *gorm.DB) error { return doc.AfterCreate(tx) }))
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
