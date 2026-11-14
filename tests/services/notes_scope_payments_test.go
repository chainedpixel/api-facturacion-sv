package services

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	ccfStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	creditNoteStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/validator/strategy"
	debitNoteStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	invoiceStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	remissionStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/validator/strategy"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/remission_note"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

func scopeTestRemission(docs []interfaces.RelatedDocument, refs ...*string) *remission_note_models.RemissionNoteModel {
	items := make([]remission_note_models.RemissionNoteItem, 0, len(refs))
	for index, ref := range refs {
		items = append(items, remission_note_models.RemissionNoteItem{Item: &models.Item{
			Number:     item.ItemNumber{Value: index + 1},
			RelatedDoc: ref,
		}})
	}
	return &remission_note_models.RemissionNoteModel{
		DTEDocument:    &models.DTEDocument{RelatedDocuments: docs},
		RemissionItems: items,
	}
}

// TestRemissionNoteItemRelatedDocs covers the strict item rule applied to the remission note.
func TestRemissionNoteItemRelatedDocs(t *testing.T) {
	test.TestMain(t)

	a, b := "DOC-A", "DOC-B"
	cases := []struct {
		name     string
		docs     []interfaces.RelatedDocument
		refs     []*string
		wantCode string
	}{
		{"no section and items in null", nil, []*string{nil, nil}, ""},
		{"every item references the doc", relatedDocsTestDocs(a), []*string{&a, &a}, ""},
		{"loose item with section rejected", relatedDocsTestDocs(a), []*string{&a, nil}, "MISSINGITEMRELATEDDOC"},
		{"two docs each referenced", relatedDocsTestDocs(a, b), []*string{&a, &b}, ""},
		{"item without ref with section rejected", relatedDocsTestDocs(a), []*string{nil}, "MISSINGITEMRELATEDDOC"},
		{"second doc not referenced", relatedDocsTestDocs(a, b), []*string{&a, &a}, "UNREFERENCEDRELATEDDOC"},
		{"ref without section", nil, []*string{&a}, "INVALIDITEMRELATEDDOC"},
		{"ref outside section", relatedDocsTestDocs(a), []*string{&a, &b}, "INVALIDITEMRELATEDDOC"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := (&remissionStrategy.RemissionNoteRelatedDocStrategy{Document: scopeTestRemission(tc.docs, tc.refs...)}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantCode, got.GetCode())
		})
	}

	t.Run("nil document is ignored", func(t *testing.T) {
		assert.Nil(t, (&remissionStrategy.RemissionNoteRelatedDocStrategy{}).Validate())
		assert.Nil(t, (&remissionStrategy.RemissionNoteRelatedDocStrategy{Document: &remission_note_models.RemissionNoteModel{}}).Validate())
	})
}

// TestRemissionNoteItemMapperKeepsDocumentNumber checks that the item document_number reaches the item related document.
func TestRemissionNoteItemMapperKeepsDocumentNumber(t *testing.T) {
	test.TestMain(t)

	number, itemType, measure := 1, 1, 59
	quantity, price, taxed := 1.0, 10.0, 10.0
	description := "Item remitido"
	reference := "DOC-A"
	request := func(documentNumber *string) []*structs.RemissionNoteItemRequest {
		return []*structs.RemissionNoteItemRequest{{
			ItemNumber: &number, ItemType: &itemType, Description: &description, Quantity: &quantity,
			UnitMeasure: &measure, UnitPrice: &price, TaxedSale: &taxed, DocumentNumber: documentNumber,
		}}
	}

	mapped, err := remission_note.MapRemissionNoteItems(request(&reference))
	require.NoError(t, err)
	require.NotNil(t, mapped[0].GetRelatedDoc())
	assert.Equal(t, reference, *mapped[0].GetRelatedDoc())

	loose, err := remission_note.MapRemissionNoteItems(request(nil))
	require.NoError(t, err)
	assert.Nil(t, loose[0].GetRelatedDoc())
}

// TestAdjustmentNotesKeepStrictItemRelatedDoc checks that NC and ND still require the related document in every item.
func TestAdjustmentNotesKeepStrictItemRelatedDoc(t *testing.T) {
	test.TestMain(t)

	creditNote, err := fixtures.BuildValidCreditNote()
	require.NoError(t, err)
	require.NotEmpty(t, creditNote.CreditItems)
	assert.Nil(t, (&creditNoteStrategy.CreditNoteRelatedDocStrategy{Document: creditNote}).Validate())
	creditNote.CreditItems[0].SetForceRelatedDoc(nil)
	got := (&creditNoteStrategy.CreditNoteRelatedDocStrategy{Document: creditNote}).Validate()
	require.NotNil(t, got)
	assert.Equal(t, "MISSINGITEMRELATEDDOC", got.GetCode())

	debitNote, err := fixtures.BuildValidDebitNote()
	require.NoError(t, err)
	require.NotEmpty(t, debitNote.DebitItems)
	assert.Nil(t, (&debitNoteStrategy.DebitNoteRelatedDocStrategy{Document: debitNote}).Validate())
	debitNote.DebitItems[0].SetForceRelatedDoc(nil)
	got = (&debitNoteStrategy.DebitNoteRelatedDocStrategy{Document: debitNote}).Validate()
	require.NotNil(t, got)
	assert.Equal(t, "MISSINGITEMRELATEDDOC", got.GetCode())
}

func scopeTestTransactions(entries ...dte.BalanceTransaction) []dte.BalanceTransaction {
	return entries
}

// TestDebitNoteBalanceLimitsOnlyNotSubject covers the debit note limit: only the non-subject sale is capped by the
// net non-subject total of the CCF minus the debit notes still in force; taxed and exempt sales are never capped.
func TestDebitNoteBalanceLimitsOnlyNotSubject(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name         string
		balance      dte.BalanceControl
		taxed        float64
		exempt       float64
		notSubject   float64
		wantRejected bool
	}{
		{"taxed above remaining balance accepted", dte.BalanceControl{OriginalTaxedAmount: 100, RemainingTaxedAmount: 10}, 500, 0, 0, false},
		{"exempt above remaining balance accepted", dte.BalanceControl{OriginalExemptAmount: 50, RemainingExemptAmount: 0}, 0, 300, 0, false},
		{"taxed on a CCF without taxed sales accepted", dte.BalanceControl{OriginalNotSubjectAmount: 20, RemainingNotSubjectAmount: 20}, 75, 0, 0, false},
		{"not subject equal to the net total accepted", dte.BalanceControl{OriginalNotSubjectAmount: 40, RemainingNotSubjectAmount: 40}, 0, 0, 40, false},
		{"not subject above the net total rejected", dte.BalanceControl{OriginalNotSubjectAmount: 40, RemainingNotSubjectAmount: 40}, 0, 0, 40.01, true},
		{"not subject on a CCF without not subject sales rejected", dte.BalanceControl{OriginalTaxedAmount: 100}, 0, 0, 1, true},
		{"previous debit notes consume the limit", dte.BalanceControl{OriginalNotSubjectAmount: 40, Transactions: scopeTestTransactions(
			dte.BalanceTransaction{TransactionType: constants.NotaDebitoElectronica, NotSubjectAmount: 30},
		)}, 0, 0, 10.01, true},
		{"remaining limit after previous debit notes accepted", dte.BalanceControl{OriginalNotSubjectAmount: 40, Transactions: scopeTestTransactions(
			dte.BalanceTransaction{TransactionType: constants.NotaDebitoElectronica, NotSubjectAmount: 30},
		)}, 0, 0, 10, false},
		{"invalidated debit notes release the limit", dte.BalanceControl{OriginalNotSubjectAmount: 40, Transactions: scopeTestTransactions(
			dte.BalanceTransaction{TransactionType: constants.NotaDebitoElectronica, NotSubjectAmount: 30},
			dte.BalanceTransaction{TransactionType: constants.DebitNoteInvalid, NotSubjectAmount: 30},
		)}, 0, 0, 40, false},
		{"credit notes do not consume the debit note limit", dte.BalanceControl{OriginalNotSubjectAmount: 40, RemainingNotSubjectAmount: 5, Transactions: scopeTestTransactions(
			dte.BalanceTransaction{TransactionType: constants.NotaCreditoElectronica, NotSubjectAmount: 35},
		)}, 0, 0, 40, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			balance := tc.balance
			repo := mocks.NewMockDTERepositoryPort(ctrl)
			repo.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "CCF-1").Return(&balance, nil)

			debitNote, err := fixtures.BuildValidDebitNote()
			require.NoError(t, err)
			require.NoError(t, debitNote.Summary.SetTotalTaxed(tc.taxed))
			require.NoError(t, debitNote.Summary.SetTotalExempt(tc.exempt))
			require.NoError(t, debitNote.Summary.SetTotalNonSubject(tc.notSubject))

			err = dte_documents.NewDTEService(repo, nil).ValidateForDebitNote(context.Background(), 1, "CCF-1", debitNote)
			if tc.wantRejected {
				require.Error(t, err)
				test.AssertErrorCode(t, err, "InvalidDebitNoteNotSubjectAmount")
				assert.NotContains(t, err.Error(), "No se encontr")
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestCreditNoteBalanceStillCapsEveryType checks that the credit note keeps capping taxed, exempt and non-subject sales.
func TestCreditNoteBalanceStillCapsEveryType(t *testing.T) {
	test.TestMain(t)
	builder := fixtures.NewDTEBuilder()

	balance := &dte.BalanceControl{RemainingTaxedAmount: 50, RemainingExemptAmount: 20, RemainingNotSubjectAmount: 10}
	cases := []struct {
		name                      string
		taxed, exempt, notSubject float64
		wantRejected              bool
	}{
		{"within every balance", 50, 20, 10, false},
		{"taxed above balance", 50.01, 0, 0, true},
		{"exempt above balance", 0, 20.01, 0, true},
		{"not subject above balance", 0, 0, 10.01, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockDTERepositoryPort(ctrl)
			repo.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "CCF-1").Return(balance, nil)

			creditNote, err := builder.BuildCreditNote()
			require.NoError(t, err)
			require.NoError(t, creditNote.Summary.SetTotalTaxed(tc.taxed))
			require.NoError(t, creditNote.Summary.SetTotalExempt(tc.exempt))
			require.NoError(t, creditNote.Summary.SetTotalNonSubject(tc.notSubject))

			err = dte_documents.NewDTEService(repo, nil).ValidateForCreditNote(context.Background(), 1, "CCF-1", creditNote)
			if tc.wantRejected {
				test.AssertErrorCode(t, err, "InvalidCreditNoteTransaction")
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestCashAndOtherConditionPayments verifies the payment rules of conditions 1 and 3 and the no-payment-due rule.
func TestCashAndOtherConditionPayments(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name      string
		dteType   string
		condition int
		total     float64
		payments  []creditPaymentSpec
		wantCode  string
	}{
		{"condition 3 with 01 and 13 summing the total", constants.FacturaElectronica, 3, 100, []creditPaymentSpec{{code: "01", amount: 40}, {code: "13", amount: 60}}, ""},
		{"condition 3 with three methods", constants.CCFElectronico, 3, 100, []creditPaymentSpec{{code: "01", amount: 20}, {code: "03", amount: 30}, {code: "13", amount: 50}}, ""},
		{"condition 3 in FSE with 01 and 13", constants.FacturaSujetoExcluidoElectronica, 3, 50, []creditPaymentSpec{{code: "01", amount: 25}, {code: "13", amount: 25}}, ""},
		{"condition 3 short by 0.01 accepted", constants.FacturaElectronica, 3, 100, []creditPaymentSpec{{code: "01", amount: 40}, {code: "13", amount: 59.99}}, ""},
		{"condition 3 short by 0.02 rejected", constants.FacturaElectronica, 3, 100, []creditPaymentSpec{{code: "01", amount: 40}, {code: "13", amount: 59.98}}, "INVALIDPAYMENTTOTAL"},
		{"condition 3 over by 0.02 rejected", constants.FacturaElectronica, 3, 100, []creditPaymentSpec{{code: "01", amount: 40}, {code: "13", amount: 60.02}}, "INVALIDPAYMENTTOTAL"},
		{"condition 3 without code rejected", constants.FacturaElectronica, 3, 100, []creditPaymentSpec{{code: "01", amount: 40}, {amount: 60}}, "PAYMENTCODEREQUIRED"},
		{"condition 1 with two methods summing the total", constants.FacturaElectronica, 1, 100, []creditPaymentSpec{{code: "01", amount: 50}, {code: "02", amount: 50}}, ""},
		{"condition 1 over by 0.01 accepted", constants.FacturaElectronica, 1, 100, []creditPaymentSpec{{code: "01", amount: 100.01}}, ""},
		{"condition 1 without code rejected", constants.CCFElectronico, 1, 100, []creditPaymentSpec{{amount: 100}}, "PAYMENTCODEREQUIRED"},
		{"total zero with condition 1 and code 99", constants.FacturaElectronica, 1, 0, []creditPaymentSpec{{code: "99", amount: 0}}, ""},
		{"total zero with condition 1 and code 99 in CCF", constants.CCFElectronico, 1, 0, []creditPaymentSpec{{code: "99", amount: 0}}, ""},
		{"total zero with condition 2 rejected", constants.FacturaElectronica, 2, 0, []creditPaymentSpec{{code: "99", amount: 0}}, "NOPAYMENTDUEREQUIRESCASHCONDITION"},
		{"total zero with condition 3 rejected", constants.FacturaElectronica, 3, 0, []creditPaymentSpec{{code: "99", amount: 0}}, "NOPAYMENTDUEREQUIRESCASHCONDITION"},
		{"total zero with code 01 rejected", constants.FacturaElectronica, 1, 0, []creditPaymentSpec{{code: "01", amount: 0}}, "NOPAYMENTDUEREQUIRESCODE99"},
		{"total zero with 99 and 01 rejected", constants.FacturaElectronica, 1, 0, []creditPaymentSpec{{code: "99", amount: 0}, {code: "01", amount: 0}}, "NOPAYMENTDUEREQUIRESCODE99"},
		{"total zero without payments rejected", constants.FacturaElectronica, 1, 0, nil, "NOPAYMENTDUEREQUIRESCODE99"},
		{"total zero in FSE with code 99", constants.FacturaSujetoExcluidoElectronica, 1, 0, []creditPaymentSpec{{code: "99", amount: 0}}, ""},
		{"total zero in FSE with code 01 accepted", constants.FacturaSujetoExcluidoElectronica, 1, 0, []creditPaymentSpec{{code: "01", amount: 0}}, ""},
		{"total zero in FSE without payments accepted", constants.FacturaSujetoExcluidoElectronica, 1, 0, nil, ""},
		{"total zero in FSE with condition 2 rejected", constants.FacturaSujetoExcluidoElectronica, 2, 0, []creditPaymentSpec{{amount: 0, term: creditTestStr("01"), period: creditTestInt(30)}}, "NOPAYMENTDUEREQUIRESCASHCONDITION"},
		{"total zero in FSE with condition 3 rejected", constants.FacturaSujetoExcluidoElectronica, 3, 0, []creditPaymentSpec{{code: "99", amount: 0}}, "NOPAYMENTDUEREQUIRESCASHCONDITION"},
		{"total zero in NC with condition 1 accepted", constants.NotaCreditoElectronica, 1, 0, nil, ""},
		{"total zero in NC with condition 2 rejected", constants.NotaCreditoElectronica, 2, 0, nil, "NOPAYMENTDUEREQUIRESCASHCONDITION"},
		{"total zero in ND with condition 3 rejected", constants.NotaDebitoElectronica, 3, 0, nil, "NOPAYMENTDUEREQUIRESCASHCONDITION"},
		{"total zero in ND with condition 1 accepted", constants.NotaDebitoElectronica, 1, 0, nil, ""},
		{"total above zero in NC with condition 2 accepted", constants.NotaCreditoElectronica, 2, 50, nil, ""},
		{"total above zero in ND with condition 3 accepted", constants.NotaDebitoElectronica, 3, 50, nil, ""},
		{"total zero in NR is not validated", constants.NotaRemisionElectronica, 2, 0, nil, ""},
		{"code 99 with total above zero keeps the sum rule", constants.FacturaElectronica, 1, 100, []creditPaymentSpec{{code: "99", amount: 100}}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := creditTestPaymentDocument(tc.dteType, tc.condition, tc.total, tc.payments...)
			got := (&strategy.PaymentTotalStrategy{Document: doc}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantCode, got.GetCode())
			assert.NotContains(t, got.Message, "No se encontr")
			assert.NotContains(t, got.Message, "%!")
		})
	}
}

// TestBalanceInFavorPayments covers the no-payment-due rule triggered by a balance in favor in FE and CCF.
func TestBalanceInFavorPayments(t *testing.T) {
	test.TestMain(t)

	summary := func(condition int, total float64, payments ...creditPaymentSpec) interfaces.Summary {
		return creditTestPaymentDocument(constants.FacturaElectronica, condition, total, payments...).Summary
	}

	assert.Nil(t, strategy.ValidateBalanceInFavorPayments(summary(1, 10, creditPaymentSpec{code: "01", amount: 10}), 0))
	assert.Nil(t, strategy.ValidateBalanceInFavorPayments(summary(2, 0), -5), "a zero total is left to PaymentTotalStrategy")
	assert.Nil(t, strategy.ValidateBalanceInFavorPayments(summary(1, 10, creditPaymentSpec{code: "99", amount: 10}), -5))
	assert.Nil(t, strategy.ValidateBalanceInFavorPayments(nil, -5))

	got := strategy.ValidateBalanceInFavorPayments(summary(1, 10, creditPaymentSpec{code: "01", amount: 10}), -5)
	require.NotNil(t, got)
	assert.Equal(t, "NOPAYMENTDUEREQUIRESCODE99", got.GetCode())

	got = strategy.ValidateBalanceInFavorPayments(summary(3, 10, creditPaymentSpec{code: "99", amount: 10}), -5)
	require.NotNil(t, got)
	assert.Equal(t, "NOPAYMENTDUEREQUIRESCASHCONDITION", got.GetCode())

	invoice, err := fixtures.BuildValidInvoice()
	require.NoError(t, err)
	assert.Nil(t, (&invoiceStrategy.InvoiceTotalsStrategy{Document: invoice}).Validate())
	invoice.InvoiceSummary.BalanceInFavor = *financial.NewValidatedAmount(-5)
	got = (&invoiceStrategy.InvoiceTotalsStrategy{Document: invoice}).Validate()
	require.NotNil(t, got)
	assert.Equal(t, "NOPAYMENTDUEREQUIRESCODE99", got.GetCode())

	ccf, err := fixtures.BuildValidCCF()
	require.NoError(t, err)
	assert.Nil(t, (&ccfStrategy.CCFTaxStrategy{Document: ccf}).Validate())
	ccf.CreditSummary.BalanceInFavor = *financial.NewValidatedAmount(-5)
	got = (&ccfStrategy.CCFTaxStrategy{Document: ccf}).Validate()
	require.NotNil(t, got)
	assert.Equal(t, "NOPAYMENTDUEREQUIRESCODE99", got.GetCode())
}
