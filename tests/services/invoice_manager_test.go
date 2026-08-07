package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

func TestInvoiceServiceCreate(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name             string
		setupInvoiceData func() (*invoice_models.InvoiceData, error)
		setupMock        func(*mocks.MockSequentialNumberManager)
		wantErr          bool
		errorCode        string
	}{
		{
			name: "Valid Invoice creation",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name: "Valid Invoice for Consumer Final",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildNaturalReceiverInvoice()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name: "Valid Invoice with credit operation",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildCreditInvoice()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name: "Valid Invoice with exempt sales only",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				for i := range invoice.InvoiceItems {
					amount := invoice.InvoiceItems[i].TaxedSale.GetValue()
					taxedZero := financial.NewValidatedAmount(0.0)
					exemptAmount := financial.NewValidatedAmount(amount)

					invoice.InvoiceItems[i].TaxedSale = *taxedZero
					invoice.InvoiceItems[i].ExemptSale = *exemptAmount
					invoice.InvoiceItems[i].IVAItem = *taxedZero
					invoice.InvoiceItems[i].Taxes = nil
				}

				totalExempt := 0.0
				for _, item := range invoice.InvoiceItems {
					totalExempt += item.ExemptSale.GetValue()
				}

				invoice.InvoiceSummary.TotalIva = *financial.NewValidatedAmount(0.0)
				invoice.InvoiceSummary.Summary.SetTotalTaxed(0.0)
				invoice.InvoiceSummary.Summary.SetTotalExempt(totalExempt)
				invoice.InvoiceSummary.Summary.SetSubtotalSales(totalExempt)
				invoice.InvoiceSummary.Summary.SetSubTotal(totalExempt)
				invoice.InvoiceSummary.Summary.SetTotalOperation(totalExempt)
				invoice.InvoiceSummary.Summary.SetTotalToPay(totalExempt)
				invoice.InvoiceSummary.Summary.SetTotalTaxes(nil)

				payments := invoice.InvoiceSummary.Summary.GetPaymentTypes()
				for i, payment := range payments {
					if i == 0 {
						payment.SetAmount(totalExempt)
					}
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name: "Valid Invoice with IVA and income retentions",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				taxedAmount := invoice.InvoiceSummary.Summary.GetTotalTaxed()
				ivaRetention := financial.NewValidatedAmount(math.Round(taxedAmount * 0.01))
				incomeRetention := financial.NewValidatedAmount(math.Round(taxedAmount * 0.05))

				invoice.InvoiceSummary.IVARetention = *ivaRetention
				invoice.InvoiceSummary.IncomeRetention = *incomeRetention

				totalOperation := invoice.InvoiceSummary.Summary.GetTotalOperation()
				newTotalToPay := totalOperation - ivaRetention.GetValue() - incomeRetention.GetValue()
				newTotalToPay = decimal.NewFromFloat(newTotalToPay).Round(2).InexactFloat64()
				invoice.InvoiceSummary.Summary.SetTotalToPay(newTotalToPay)

				payments := invoice.InvoiceSummary.Summary.GetPaymentTypes()
				for i, payment := range payments {
					if i == 0 {
						payment.SetAmount(newTotalToPay)
					}
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name: "Valid Invoice with related documents",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				relatedDoc1 := models.RelatedDocument{}
				relatedDoc1.SetGenerationType(constants.ElectronicDocument)
				relatedDoc1.SetDocumentNumber("001BEDAD-93F3-4F49-85D9-1E3618425F6B")
				relatedDoc1.SetDocumentType(constants.NotaRemisionElectronica)
				relatedDoc1.SetEmissionDate(utils.TimeNow())

				relatedDoc2 := models.RelatedDocument{}
				relatedDoc2.SetGenerationType(constants.ElectronicDocument)
				relatedDoc2.SetDocumentNumber("001BEDAD-93F3-4F49-85D9-1E3618425F6C")
				relatedDoc2.SetDocumentType(constants.NotaRemisionElectronica)
				relatedDoc2.SetEmissionDate(utils.TimeNow())

				invoiceData := fixtures.BuildAsInvoiceData(invoice)
				invoiceData.RelatedDocs = []models.RelatedDocument{relatedDoc1, relatedDoc2}

				for i := range invoiceData.Items {
					docRef := "001BEDAD-93F3-4F49-85D9-1E3618425F6B"
					invoiceData.Items[i].SetRelatedDoc(&docRef)
				}

				return invoiceData, nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name: "Valid Invoice with electronic payment",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				builder := fixtures.NewInvoiceBuilder()
				builder.AddIdentification().
					AddIssuer().
					AddReceiver().
					AddItems().
					AddSummaryWithElectronicPayment()

				invoice, err := builder.Build()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name: "Error - Invoice with InvalidMixedSalesWithExempt",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				exemptAmount, _ := financial.NewAmount(100.0)
				invoice.InvoiceItems[0].ExemptSale = *exemptAmount

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "MixedSalesTypesNotAllowed",
		},
		{
			name: "Error - Invoice with InvalidMixedSalesWithNonSubject",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				nonSubjectAmount, _ := financial.NewAmount(100.0)
				invoice.InvoiceItems[0].NonSubjectSale = *nonSubjectAmount

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "MixedSalesTypesNotAllowed",
		},
		{
			name: "Error - Invoice with InvalidMixedSalesWithNonTaxed",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				nonTaxedAmount, _ := financial.NewAmount(100.0)
				invoice.InvoiceItems[0].NonTaxed = *nonTaxedAmount

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "InvalidMixedSalesWithNonTaxed",
		},
		{
			name: "Error - Invoice with InvalidIVAItemCalculation",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				incorrectIva, _ := financial.NewAmount(50.0)
				invoice.InvoiceItems[0].IVAItem = *incorrectIva

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "InvalidIVAItemCalculation",
		},
		{
			name: "Error - Invoice with DiscountExceedsSubtotal",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				subtotal := invoice.InvoiceSummary.Summary.GetSubTotal()
				invoice.InvoiceSummary.TaxedDiscount = *financial.NewValidatedAmount(subtotal * 2)

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "DiscountExceedsSubtotal",
		},
		{
			name: "Error - Invoice with InvalidSubTotalCalculation",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				incorrectSubtotal := invoice.InvoiceSummary.Summary.GetSubtotalSales() -
					invoice.InvoiceSummary.TaxedDiscount.GetValue() -
					invoice.InvoiceSummary.Summary.GetExemptDiscount() -
					invoice.InvoiceSummary.Summary.GetNonSubjectDiscount() + 50.0

				invoice.InvoiceSummary.Summary.SetSubTotal(incorrectSubtotal)

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "InvalidSubTotalCalculation",
		},
		{
			name: "Error - Invoice with InvalidMonetaryAmount",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				amount := 100.123
				invoice.InvoiceSummary.Summary.SetTotalToPay(amount)

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "InvalidMonetaryAmount",
		},
		{
			name: "Error - Invoice with ExceededRelatedDocsLimit",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
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

				invoiceData := fixtures.BuildAsInvoiceData(invoice)
				invoiceData.RelatedDocs = relatedDocs

				for i := range invoiceData.Items {
					docRef := "DTE-01-C0020000-000000000000001"
					invoiceData.Items[i].SetRelatedDoc(&docRef)
				}

				return invoiceData, nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "ExceededRelatedDocsLimit",
		},
		{
			name: "Error - Invoice with invalid total to pay",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				invoice.InvoiceSummary.Summary.SetTotalToPay(1500.0)

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {},
			wantErr:   true,
			errorCode: "InvalidTotalToPayCalculation",
		},
		{
			name: "Error - Failed to generate control number",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("", shared_error.NewGeneralServiceError("SequentialNumberManager", "GetNextControlNumber", "Failed to generate control number", nil))
			},
			wantErr:   true,
			errorCode: "Failed to generate control number",
		},
		{
			name: "Error - Failed to set control number",
			setupInvoiceData: func() (*invoice_models.InvoiceData, error) {
				invoice, err := fixtures.BuildValidInvoice()
				if err != nil {
					return nil, err
				}

				return fixtures.BuildAsInvoiceData(invoice), nil
			},
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F001000001-00A0000012345", nil)
			},
			wantErr:   true,
			errorCode: "InvalidPattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoiceData, err := tt.setupInvoiceData()
			if err != nil {
				t.Fatalf("Error preparing test data: %v", err)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSeqNumberManager := mocks.NewMockSequentialNumberManager(ctrl)
			tt.setupMock(mockSeqNumberManager)

			service := invoice.NewInvoiceService(mockSeqNumberManager)

			result, err := service.Create(context.Background(), invoiceData, 1)

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

				invoiceDoc, ok := result.(*invoice_models.ElectronicInvoice)
				assert.True(t, ok, "Result should be an ElectronicInvoice")
				assert.NotNil(t, invoiceDoc.Identification)
				assert.NotNil(t, invoiceDoc.Issuer)
				assert.NotNil(t, invoiceDoc.Receiver)
				assert.NotEmpty(t, invoiceDoc.InvoiceItems)
				assert.NotNil(t, invoiceDoc.InvoiceSummary.Summary)

				assert.NotEmpty(t, invoiceDoc.Identification.GetControlNumber())
				assert.NotEmpty(t, invoiceDoc.Identification.GetGenerationCode())
			}
		})
	}
}

func TestInvoiceServiceCreateWithDifferentOperationConditions(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name               string
		operationCondition int
		setupMock          func(*mocks.MockSequentialNumberManager)
		wantErr            bool
	}{
		{
			name:               "Valid Invoice with cash operation",
			operationCondition: constants.Cash,
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
		{
			name:               "Valid Invoice with credit operation",
			operationCondition: constants.Credit,
			setupMock: func(mock *mocks.MockSequentialNumberManager) {
				mock.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaElectronica,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return("DTE-01-F0010001-000000000012345", nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var invoiceModel *invoice_models.ElectronicInvoice
			var err error

			if tt.operationCondition == constants.Credit {
				invoiceModel, err = fixtures.BuildCreditInvoice()
				if err != nil {
					t.Fatalf("Error building Invoice: %v", err)
				}
			} else {
				invoiceModel, err = fixtures.BuildValidInvoice()
				if err != nil {
					t.Fatalf("Error building Invoice: %v", err)
				}
			}

			invoiceData := fixtures.BuildAsInvoiceData(invoiceModel)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSeqNumberManager := mocks.NewMockSequentialNumberManager(ctrl)
			tt.setupMock(mockSeqNumberManager)

			service := invoice.NewInvoiceService(mockSeqNumberManager)

			result, err := service.Create(context.Background(), invoiceData, 1)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				invoiceDoc, ok := result.(*invoice_models.ElectronicInvoice)
				assert.True(t, ok)
				assert.Equal(t, tt.operationCondition, invoiceDoc.Summary.GetOperationCondition())
			}
		})
	}
}

func TestInvoiceServiceCreateWithMixedItemTypes(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSeqNumberManager := mocks.NewMockSequentialNumberManager(ctrl)
	mockSeqNumberManager.EXPECT().GetNextControlNumber(
		gomock.Any(),
		constants.FacturaElectronica,
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return("DTE-01-F0010001-000000000012345", nil)

	service := invoice.NewInvoiceService(mockSeqNumberManager)

	builder := fixtures.NewInvoiceBuilder()
	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddItems().
		AddSummary()

	invoiceModel, err := builder.Build()
	if err != nil {
		t.Fatalf("Error building Invoice: %v", err)
	}

	invoiceData := fixtures.BuildAsInvoiceData(invoiceModel)
	result, err := service.Create(context.Background(), invoiceData, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	invoiceDoc, ok := result.(*invoice_models.ElectronicInvoice)
	assert.True(t, ok)

	foundProduct := false
	foundService := false

	for _, item := range invoiceDoc.InvoiceItems {
		if item.GetType() == constants.Producto {
			foundProduct = true
		} else if item.GetType() == constants.Servicio {
			foundService = true
		}
	}

	assert.True(t, foundProduct, "Should have at least one Product type item")
	assert.True(t, foundService, "Should have at least one Service type item")
}

func TestInvoiceServiceCreateWithElectronicPayment(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSeqNumberManager := mocks.NewMockSequentialNumberManager(ctrl)
	mockSeqNumberManager.EXPECT().GetNextControlNumber(
		gomock.Any(),
		constants.FacturaElectronica,
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return("DTE-01-F0010001-000000000012345", nil)

	service := invoice.NewInvoiceService(mockSeqNumberManager)

	builder := fixtures.NewInvoiceBuilder()
	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddItems().
		AddSummaryWithElectronicPayment()

	invoiceModel, err := builder.Build()
	if err != nil {
		t.Fatalf("Error building Invoice: %v", err)
	}

	invoiceData := fixtures.BuildAsInvoiceData(invoiceModel)
	result, err := service.Create(context.Background(), invoiceData, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	invoiceDoc, ok := result.(*invoice_models.ElectronicInvoice)
	assert.True(t, ok)
	assert.NotNil(t, invoiceDoc.InvoiceSummary.ElectronicPaymentNumber)
	assert.NotEmpty(t, *invoiceDoc.InvoiceSummary.ElectronicPaymentNumber)
}
