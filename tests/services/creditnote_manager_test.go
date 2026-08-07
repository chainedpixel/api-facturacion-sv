package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

func TestCreditNoteServiceCreate(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name                string
		setupCreditNoteData func() (*credit_note_models.CreditNoteInput, error)
		setupMock           func(*mocks.MockSequentialNumberManager, *mocks.MockDTEManager)
		wantErr             bool
		errorCode           string
	}{
		{
			name: "Valid CreditNote creation",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "Valid CreditNote with company receptor",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildCreditNoteWithCompanyReceiver()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "Valid CreditNote with credit operation",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildCreditNoteWithCreditOperation()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "Valid CreditNote with exempt sales only",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				for i := range creditNote.CreditItems {
					amount := creditNote.CreditItems[i].TaxedSale.GetValue()
					taxedZero := financial.NewValidatedAmount(0.0)
					exemptAmount := financial.NewValidatedAmount(amount)

					creditNote.CreditItems[i].TaxedSale = *taxedZero
					creditNote.CreditItems[i].ExemptSale = *exemptAmount
					creditNote.CreditItems[i].Taxes = nil
				}

				totalExempt := 0.0
				for _, item := range creditNote.CreditItems {
					totalExempt += item.ExemptSale.GetValue()
				}

				creditNote.CreditSummary.Summary.SetTotalTaxed(0.0)
				creditNote.CreditSummary.Summary.SetTotalExempt(totalExempt)
				creditNote.CreditSummary.Summary.SetSubtotalSales(totalExempt)
				creditNote.CreditSummary.Summary.SetSubTotal(totalExempt)
				creditNote.CreditSummary.Summary.SetTotalOperation(totalExempt)
				creditNote.CreditSummary.Summary.SetTotalToPay(totalExempt)
				creditNote.CreditSummary.Summary.SetTotalTaxes(nil)

				payments := creditNote.CreditSummary.Summary.GetPaymentTypes()
				for i, payment := range payments {
					if i == 0 {
						payment.SetAmount(totalExempt)
					}
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "Valid CreditNote with retentions",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				taxedAmount := creditNote.CreditSummary.Summary.GetTotalTaxed()
				ivaRetention := financial.NewValidatedAmount(taxedAmount * 0.01)
				incomeRetention := financial.NewValidatedAmount(taxedAmount * 0.05)

				creditNote.CreditSummary.IVARetention = *ivaRetention
				creditNote.CreditSummary.IncomeRetention = *incomeRetention

				totalOperation := creditNote.CreditSummary.Summary.GetTotalOperation()
				newTotalToPay := totalOperation - ivaRetention.GetValue() - incomeRetention.GetValue()
				newTotalToPay = decimal.NewFromFloat(newTotalToPay).Round(2).InexactFloat64()
				creditNote.CreditSummary.Summary.SetTotalToPay(newTotalToPay)
				creditNote.CreditSummary.Summary.SetTotalOperation(newTotalToPay)

				payments := creditNote.CreditSummary.Summary.GetPaymentTypes()
				for i, payment := range payments {
					if i == 0 {
						payment.SetAmount(newTotalToPay)
					}
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "Error - CreditNote without related documents",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNoteInput := fixtures.BuildAsCreditNoteInput(creditNote)
				creditNoteInput.RelatedDocs = nil

				return creditNoteInput, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
			},
			wantErr:   true,
			errorCode: "NoRelatedDocs",
		},
		{
			name: "Error - CreditNote with related document not found",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, fmt.Errorf("document not found"))
			},
			wantErr:   true,
			errorCode: "document not found",
		},
		{
			name: "Error - CreditNote with related document not received",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil)

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentRejected, nil)
			},
			wantErr:   true,
			errorCode: "DocumentNotReceived",
		},
		{
			name: "Error - CreditNote with NIT mismatch in related document",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"1111111111111"}}`,
					},
				}, nil)

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil)
			},
			wantErr:   true,
			errorCode: "NotMatchingReceiverNIT",
		},
		{
			name: "Error - CreditNote with MixedSalesTypes",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				exemptAmount, _ := financial.NewAmount(100.0)
				creditNote.CreditItems[0].ExemptSale = *exemptAmount

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "MixedSalesTypesNotAllowed",
		},
		{
			name: "Error - CreditNote with invalid tax calculation",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				for _, t := range creditNote.CreditSummary.Summary.GetTotalTaxes() {
					tax, ok := t.(*models.Tax)
					if ok && tax.GetCode() == constants.TaxIVA {
						tax.SetValue(50.0)
						break
					}
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidTaxCalculation",
		},
		{
			name: "Error - CreditNote with InvalidPerceptionAmount",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				taxedAmount := creditNote.CreditSummary.Summary.GetTotalTaxed()
				incorrectPerception := financial.NewValidatedAmount(taxedAmount * 0.02)
				creditNote.CreditSummary.IVAPerception = *incorrectPerception

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidPerceptionAmount",
		},
		{
			name: "Error - CreditNote with missing item related doc",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNote.CreditItems[0].SetForceRelatedDoc(nil)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "MissingItemRelatedDoc",
		},
		{
			name: "Error - CreditNote with invalid item related doc",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				invalidRef := "INVALID-REF-00000000"
				creditNote.CreditItems[0].SetRelatedDoc(&invalidRef)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidItemRelatedDoc",
		},
		{
			name: "Error - CreditNote with exceeded related docs limit",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				relatedDocs := make([]models.RelatedDocument, 51)
				for i := 0; i < 51; i++ {
					doc := models.RelatedDocument{}
					docNum := fmt.Sprintf("DTE-01-C0020000-00000000000000%d", i)
					doc.SetDocumentNumber(docNum)
					doc.SetDocumentType(constants.FacturaElectronica)
					doc.SetEmissionDate(utils.TimeNow())
					relatedDocs[i] = doc
				}

				creditNoteInput := fixtures.BuildAsCreditNoteInput(creditNote)
				creditNoteInput.RelatedDocs = relatedDocs

				return creditNoteInput, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "ExceededRelatedDocsLimit",
		},
		{
			name: "Error - CreditNote with invalid related doc type",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				invalidType := constants.FacturaElectronica
				creditNote.RelatedDocuments[0].SetDocumentType(invalidType)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidRelatedDocTypeForCreditNote",
		},
		{
			name: "Error - CreditNote with invalid unit price zero",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNote.CreditItems[0].Item.SetUnitPrice(0)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidUnitPriceZero",
		},
		{
			name: "Error - CreditNote with invalid unit measure for type 4",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNote.CreditItems[0].Item.SetType(constants.Impuesto)
				creditNote.CreditItems[0].Item.SetUnitMeasure(58)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidUnitMeasure",
		},
		{
			name: "Error - CreditNote with invalid tax rules for type 4",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNote.CreditItems[0].Item.SetType(constants.Impuesto)
				creditNote.CreditItems[0].Item.SetUnitMeasure(99)
				creditNote.CreditItems[0].Item.SetTaxes([]string{constants.TaxIVA, constants.TaxTourism})

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidTaxRulesCCF",
		},
		{
			name: "Error - CreditNote with invalid discount exceeding subtotal",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				subtotal := creditNote.CreditSummary.Summary.GetSubTotal()
				creditNote.CreditSummary.TaxedDiscount = *financial.NewValidatedAmount(subtotal * 2)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "DiscountExceedsSubtotal",
		},
		{
			name: "Error - CreditNote with inconsistent total taxed",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNote.CreditSummary.Summary.SetTotalTaxed(2000.0)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidTotalTaxed",
		},
		{
			name: "Error - CreditNote with missing taxes",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNote.CreditSummary.Summary.SetTotalTaxes(nil)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "MissingTaxes",
		},
		{
			name: "Error - CreditNote with invalid subtotal calculation",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				incorrectSubtotal := creditNote.CreditSummary.Summary.GetSubtotalSales() -
					creditNote.CreditSummary.TaxedDiscount.GetValue() -
					creditNote.CreditSummary.Summary.GetExemptDiscount() -
					creditNote.CreditSummary.Summary.GetNonSubjectDiscount() + 50.0

				creditNote.CreditSummary.Summary.SetSubTotal(incorrectSubtotal)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidSubTotalCalculation",
		},
		{
			name: "Error - CreditNote with invalid monetary amount",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				creditNote.CreditSummary.IVAPerception = *financial.NewValidatedAmount(9.505)

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr:   true,
			errorCode: "InvalidMonetaryAmount",
		},
		{
			name: "Error - Failed to generate control number",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()

				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("", shared_error.NewGeneralServiceError("SequentialNumberManager", "GetNextControlNumber", "Failed to generate control number", nil))
			},
			wantErr:   true,
			errorCode: "Failed to generate control number",
		},
		{
			name: "Error - Failed to set control number (invalid format)",
			setupCreditNoteData: func() (*credit_note_models.CreditNoteInput, error) {
				creditNote, err := fixtures.BuildValidCreditNote()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsCreditNoteInput(creditNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()

				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-000-INVALID", nil)
			},
			wantErr:   true,
			errorCode: "InvalidPattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creditNoteData, err := tt.setupCreditNoteData()
			if err != nil {
				t.Fatalf("Error preparing test data: %v", err)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSeqNumberManager := mocks.NewMockSequentialNumberManager(ctrl)
			mockDTEManager := mocks.NewMockDTEManager(ctrl)
			tt.setupMock(mockSeqNumberManager, mockDTEManager)

			service := credit_note.NewCreditNoteService(mockSeqNumberManager, mockDTEManager)

			result, err := service.Create(context.Background(), creditNoteData, 1)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					var dteErr *dte_errors.DTEError
					var serviceErr *shared_error.ServiceError

					if errors.As(err, &dteErr) {
						assert.Contains(t, dteErr.Error(), tt.errorCode, "Error message should contain expected code")
					} else if errors.As(err, &serviceErr) {
						assert.Contains(t, serviceErr.Error(), tt.errorCode, "Error message should contain expected code")
					} else {
						assert.Contains(t, err.Error(), tt.errorCode, "Error message should contain expected code")
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				creditNoteDoc, ok := result.(*credit_note_models.CreditNoteModel)
				assert.True(t, ok, "Result should be a CreditNoteModel")
				assert.NotNil(t, creditNoteDoc.Identification)
				assert.NotNil(t, creditNoteDoc.Issuer)
				assert.NotNil(t, creditNoteDoc.Receiver)
				assert.NotEmpty(t, creditNoteDoc.CreditItems)
				assert.NotNil(t, creditNoteDoc.CreditSummary.Summary)
				assert.NotEmpty(t, creditNoteDoc.RelatedDocuments)

				assert.NotEmpty(t, creditNoteDoc.Identification.GetControlNumber())
				assert.NotEmpty(t, creditNoteDoc.Identification.GetGenerationCode())
			}
		})
	}
}

func TestCreditNoteServiceCreateWithDifferentOperationConditions(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name               string
		operationCondition int
		setupMock          func(*mocks.MockSequentialNumberManager, *mocks.MockDTEManager)
		wantErr            bool
	}{
		{
			name:               "Valid CreditNote with cash operation",
			operationCondition: constants.Cash,
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name:               "Valid CreditNote with credit operation",
			operationCondition: constants.Credit,
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaCreditoElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					CreatedAt: utils.TimeNow(),
					Details: &dte.DTEDetails{
						JSONData: `{"receptor":{"nit":"06140101901011"}}`,
					},
				}, nil).AnyTimes()

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(constants.DocumentReceived, nil).AnyTimes()

				mockDTE.EXPECT().ValidateForCreditNote(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var creditNoteModel *credit_note_models.CreditNoteModel
			var err error

			if tt.operationCondition == constants.Credit {
				creditNoteModel, err = fixtures.BuildCreditNoteWithCreditOperation()
				if err != nil {
					t.Fatalf("Error building CreditNote: %v", err)
				}
			} else {
				creditNoteModel, err = fixtures.BuildValidCreditNote()
				if err != nil {
					t.Fatalf("Error building CreditNote: %v", err)
				}
			}

			creditNoteData := fixtures.BuildAsCreditNoteInput(creditNoteModel)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSeqNumberManager := mocks.NewMockSequentialNumberManager(ctrl)
			mockDTEManager := mocks.NewMockDTEManager(ctrl)
			tt.setupMock(mockSeqNumberManager, mockDTEManager)

			service := credit_note.NewCreditNoteService(mockSeqNumberManager, mockDTEManager)

			result, err := service.Create(context.Background(), creditNoteData, 1)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				creditNoteDoc, ok := result.(*credit_note_models.CreditNoteModel)
				assert.True(t, ok)
				assert.Equal(t, tt.operationCondition, creditNoteDoc.Summary.GetOperationCondition())
			}
		})
	}
}
