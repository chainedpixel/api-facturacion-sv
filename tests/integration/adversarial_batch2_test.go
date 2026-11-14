package integration_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

func adversarialBaseInvoiceRequest() *structs.CreateInvoiceRequest {
	req := fixtures.CreateDefaultInvoiceRequest()
	for i := range req.Items {
		req.Items[i].Taxes = nil
		req.Items[i].IVAItem = 5.75
	}
	req.Summary.Taxes = nil
	req.Summary.TotalIVA = 11.5
	return req
}

const adversarialRelatedNumber = "001BEDAD-93F3-4F49-85D9-1E3618425F6B"

func adversarialInvoiceRequest(docType string, itemRefs ...*string) *structs.CreateInvoiceRequest {
	req := adversarialBaseInvoiceRequest()
	req.RelatedDocs = []structs.RelatedDocRequest{{
		DocumentType:   docType,
		GenerationType: constants.ElectronicDocument,
		DocumentNumber: adversarialRelatedNumber,
		EmissionDate:   utils.TimeNow().Format("2006-01-02"),
	}}
	for i := range req.Items {
		if i < len(itemRefs) {
			req.Items[i].RelatedDoc = itemRefs[i]
		}
	}
	return req
}

func adversarialCreateInvoice(t *testing.T, req *structs.CreateInvoiceRequest) error {
	t.Helper()
	data, err := request_mapper.NewInvoiceMapper().MapToInvoiceData(req, fixtures.CreateDefaultIssuer())
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	seq := mocks.NewMockSequentialNumberManager(ctrl)
	seq.EXPECT().GetNextControlNumber(gomock.Any(), constants.FacturaElectronica, gomock.Any(), gomock.Any(), gomock.Any()).
		Return("DTE-01-F0010001-000000000012345", nil).AnyTimes()

	_, err = invoice.NewInvoiceService(seq).Create(context.Background(), data, 1)
	return err
}

func adversarialErrorText(err error) string {
	var dteErr *dte_errors.DTEError
	if errors.As(err, &dteErr) {
		text := dteErr.Error()
		for _, business := range dteErr.BusinessErrors {
			text += " " + business.GetCode()
		}
		return text
	}
	return err.Error()
}

// TestAdversarialInvoiceRelatedDocsMapperToValidator drives the FE request mapper and service through the rules validator.
func TestAdversarialInvoiceRelatedDocsMapperToValidator(t *testing.T) {
	test.TestMain(t)

	ref := adversarialRelatedNumber
	other := "OTHER-DOC"

	t.Run("baseline without related docs is accepted", func(t *testing.T) {
		require.NoError(t, adversarialCreateInvoice(t, adversarialBaseInvoiceRequest()))
	})

	cases := []struct {
		name    string
		req     func() *structs.CreateInvoiceRequest
		wantErr string
	}{
		{"type 04 with all items referencing it", func() *structs.CreateInvoiceRequest {
			return adversarialInvoiceRequest(constants.NotaRemisionElectronica, &ref, &ref)
		}, ""},
		{"type 09 with all items referencing it", func() *structs.CreateInvoiceRequest {
			return adversarialInvoiceRequest(constants.DocContableLiquidacionElectronico, &ref, &ref)
		}, ""},
		{"type 03 is rejected", func() *structs.CreateInvoiceRequest {
			return adversarialInvoiceRequest(constants.CCFElectronico, &ref, &ref)
		}, "INVALIDRELATEDDOCDTETYPE"},
		{"item without related doc is rejected even when another item references it", func() *structs.CreateInvoiceRequest {
			return adversarialInvoiceRequest(constants.NotaRemisionElectronica, &ref, nil)
		}, "MISSINGITEMRELATEDDOC"},
		{"no item referencing the related doc is rejected", func() *structs.CreateInvoiceRequest {
			return adversarialInvoiceRequest(constants.NotaRemisionElectronica, nil, nil)
		}, "MISSINGITEMRELATEDDOC"},
		{"item referencing a doc without related_docs section is rejected", func() *structs.CreateInvoiceRequest {
			req := adversarialBaseInvoiceRequest()
			req.Items[0].RelatedDoc = &ref
			return req
		}, "INVALIDITEMRELATEDDOC"},
		{"item referencing unknown doc is rejected", func() *structs.CreateInvoiceRequest {
			return adversarialInvoiceRequest(constants.NotaRemisionElectronica, &ref, &other)
		}, "INVALIDITEMRELATEDDOC"},
		{"empty related_docs array behaves like none", func() *structs.CreateInvoiceRequest {
			req := adversarialBaseInvoiceRequest()
			req.RelatedDocs = []structs.RelatedDocRequest{}
			return req
		}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := adversarialCreateInvoice(t, tc.req())
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, strings.ToUpper(adversarialErrorText(err)), tc.wantErr)
		})
	}
}

func adversarialMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB, mock
}

type adversarialCapture struct{ args *[]interface{} }

func (c adversarialCapture) Match(value driver.Value) bool {
	*c.args = append(*c.args, value)
	return true
}

// TestAdversarialNetBalanceAfterCreateThenCreditNoteValidation persists the initial balance through AfterCreate,
// feeds the persisted amounts to ValidateForCreditNote and checks the credit note boundaries.
func TestAdversarialNetBalanceAfterCreateThenCreditNoteValidation(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name       string
		resumen    string
		creditNote float64
		wantErr    bool
	}{
		{"10 percent discount credit note above net", `{"resumen":{"totalGravada":100,"descuGravada":10}}`, 95, true},
		{"10 percent discount credit note at net", `{"resumen":{"totalGravada":100,"descuGravada":10}}`, 90, false},
		{"one cent above net", `{"resumen":{"totalGravada":100,"descuGravada":10}}`, 90.01, true},
		{"old document without discounts allows full total", `{"resumen":{"totalGravada":100}}`, 100, false},
		{"discount equal to total allows nothing", `{"resumen":{"totalGravada":100,"descuGravada":100}}`, 0.01, true},
		{"discount above total clamps to zero", `{"resumen":{"totalGravada":100,"descuGravada":150}}`, 0.01, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := adversarialMockDB(t)
			var args []interface{}

			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO `dte_balance_control`").
				WithArgs(adversarialCapture{&args}, adversarialCapture{&args}, adversarialCapture{&args}, adversarialCapture{&args},
					adversarialCapture{&args}, adversarialCapture{&args}, adversarialCapture{&args}, adversarialCapture{&args}).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			doc := &db_models.DTEDocument{
				DocumentID: "CCF-1",
				BranchID:   1,
				Document:   &db_models.DTEDetails{DTEType: constants.CCFElectronico, JSONData: tc.resumen},
			}
			require.NoError(t, gormDB.Transaction(func(tx *gorm.DB) error { return doc.AfterCreate(tx) }))
			require.Len(t, args, 8)

			persisted := &dte.BalanceControl{
				RemainingTaxedAmount:      args[5].(float64),
				RemainingExemptAmount:     args[6].(float64),
				RemainingNotSubjectAmount: args[7].(float64),
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockDTERepositoryPort(ctrl)
			repo.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "CCF-1").Return(persisted, nil)

			creditNote, err := fixtures.NewDTEBuilder().BuildCreditNote()
			require.NoError(t, err)
			require.NoError(t, creditNote.Summary.SetTotalTaxed(tc.creditNote))
			require.NoError(t, creditNote.Summary.SetTotalExempt(0))
			require.NoError(t, creditNote.Summary.SetTotalNonSubject(0))

			err = dte_documents.NewDTEService(repo, nil).ValidateForCreditNote(context.Background(), 1, "CCF-1", creditNote)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, strings.ToUpper(err.Error()), "INVALIDCREDITNOTETRANSACTION")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
