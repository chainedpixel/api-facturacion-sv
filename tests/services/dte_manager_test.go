package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// createMockResponse creates a fictitious DTE document for testing
func createMockInvoiceResponse() *structs.InvoiceDTEResponse {
	return &structs.InvoiceDTEResponse{
		Identificacion: &structs.DTEIdentification{
			TipoDte: constants.FacturaElectronica,
		},
		Apendice: []structs.DTEApendice{},
	}
}

func createMockResponseForType(dteType string) interface{} {
	switch dteType {
	case constants.FacturaElectronica:
		return &structs.InvoiceDTEResponse{
			Identificacion: &structs.DTEIdentification{TipoDte: dteType},
			Apendice:       []structs.DTEApendice{},
		}
	case constants.CCFElectronico:
		return &structs.CCFDTEResponse{
			Identificacion: &structs.DTEIdentification{TipoDte: dteType},
			Apendice:       []structs.DTEApendice{},
		}
	case constants.NotaCreditoElectronica:
		return &structs.CreditNoteDTEResponse{
			Identificacion: &structs.DTEIdentification{TipoDte: dteType},
			Apendice:       []structs.DTEApendice{},
		}
	case constants.NotaDebitoElectronica:
		return &structs.DebitNoteDTEResponse{
			Identificacion: &structs.DTEIdentification{TipoDte: dteType},
			Apendice:       []structs.DTEApendice{},
		}
	case constants.ComprobanteRetencionElectronico:
		return &structs.RetentionDTEResponse{
			Identificacion: &structs.DTEIdentification{TipoDte: dteType},
			Apendice:       []structs.DTEApendice{},
		}
	case constants.NotaRemisionElectronica:
		return &structs.MHRemissionNote{
			Identification: &structs.DTEIdentification{TipoDte: dteType},
			Appendix:       []structs.DTEApendice{},
		}
	case constants.FacturaSujetoExcluidoElectronica:
		return &structs.FSEDTEResponse{
			Identificacion: structs.DTEIdentification{TipoDte: dteType},
		}
	}
	return nil
}

// TestDTEServiceCreate tests the Create function of the DTE service
func TestDTEServiceCreate(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name           string
		document       interface{}
		transmission   string
		status         string
		receptionStamp *string
		setupMock      func(*mocks.MockDTERepositoryPort)
		wantErr        bool
		errorCode      string
	}{
		{
			name:           "Valid DTE creation without contingency",
			document:       createMockInvoiceResponse(),
			transmission:   constants.TransmissionNormal,
			status:         constants.DocumentReceived,
			receptionStamp: utils.ToStringPointer("TEST-STAMP-123"),
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().Create(gomock.Any(), gomock.Any(), constants.TransmissionNormal, constants.DocumentReceived, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:           "Valid DTE creation with contingency",
			document:       createMockInvoiceResponse(),
			transmission:   constants.TransmissionContingency,
			status:         constants.DocumentPending,
			receptionStamp: nil,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, nil).Return(nil)
			},
			wantErr: false,
		},
		{
			name:           "Failed to create DTE in database",
			document:       createMockInvoiceResponse(),
			transmission:   constants.TransmissionNormal,
			status:         constants.TransmissionNormal,
			receptionStamp: utils.ToStringPointer("TEST-STAMP-123"),
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(shared_error.NewFormattedGeneralServiceError("Repository", "Create", "FailedToCreateDTE"))
			},
			wantErr:   true,
			errorCode: "FailedToCreateDTE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			err := service.Create(context.Background(), tt.document, tt.transmission, tt.status, tt.receptionStamp)

			if tt.wantErr {
				assert.Error(t, err)
				test.AssertErrorCode(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDTEServiceGenerateBalanceTransaction tests the GenerateBalanceTransaction function
func TestDTEServiceGenerateBalanceTransaction(t *testing.T) {
	test.TestMain(t)

	mockDoc := createMockInvoiceResponse()
	mockDoc.Resumen = &structs.InvoiceSummary{
		TotalGravada: 100.0,
		TotalExenta:  50.0,
		TotalNoSuj:   25.0,
	}

	tests := []struct {
		name            string
		branchID        uint
		transactionType string
		originalDTE     string
		adjustmentDTE   string
		document        interface{}
		setupMock       func(*mocks.MockDTERepositoryPort)
		wantErr         bool
		errorCode       string
	}{
		{
			name:            "Valid balance transaction generation",
			branchID:        1,
			transactionType: constants.NotaCreditoElectronica,
			originalDTE:     "ORIGINAL-DTE-123",
			adjustmentDTE:   "ADJUSTMENT-DTE-456",
			document:        mockDoc,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GenerateBalanceTransaction(gomock.Any(), uint(1), "ORIGINAL-DTE-123", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:            "Failed to generate balance transaction",
			branchID:        1,
			transactionType: constants.NotaCreditoElectronica,
			originalDTE:     "ORIGINAL-DTE-123",
			adjustmentDTE:   "ADJUSTMENT-DTE-456",
			document:        mockDoc,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GenerateBalanceTransaction(gomock.Any(), uint(1), "ORIGINAL-DTE-123", gomock.Any()).
					Return(shared_error.NewFormattedGeneralServiceError("Repository", "GenerateBalanceTransaction", "FailedToGenerateBalanceTransaction"))
			},
			wantErr:   true,
			errorCode: "FailedToGenerateBalanceTransaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			err := service.GenerateBalanceTransaction(context.Background(), tt.branchID, tt.transactionType, tt.originalDTE, tt.adjustmentDTE, tt.document)

			if tt.wantErr {
				assert.Error(t, err)
				test.AssertErrorCode(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDTEServiceValidateForCreditNote tests the ValidateForCreditNote function
func TestDTEServiceValidateForCreditNote(t *testing.T) {
	test.TestMain(t)
	builder := fixtures.NewDTEBuilder()

	validBalance := &dte.BalanceControl{
		RemainingTaxedAmount:      100.0,
		RemainingExemptAmount:     100.0,
		RemainingNotSubjectAmount: 100.0,
	}

	tests := []struct {
		name          string
		branchID      uint
		originalDTE   string
		setupDocument func() (*credit_note_models.CreditNoteModel, error)
		setupMock     func(*mocks.MockDTERepositoryPort)
		wantErr       bool
		errorCode     string
	}{
		{
			name:        "Valid credit note",
			branchID:    1,
			originalDTE: "ORIGINAL-DTE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "ORIGINAL-DTE-123").Return(validBalance, nil)
			},
			setupDocument: func() (*credit_note_models.CreditNoteModel, error) {
				doc, err := builder.BuildCreditNote()
				if err != nil {
					return nil, err
				}

				err = doc.Summary.SetTotalTaxed(50.0)
				err = doc.Summary.SetTotalExempt(25.0)
				err = doc.Summary.SetTotalNonSubject(10.0)

				return doc, err
			},
			wantErr: false,
		},
		{
			name:        "Exceeds taxed amount",
			branchID:    1,
			originalDTE: "ORIGINAL-DTE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "ORIGINAL-DTE-123").Return(validBalance, nil)
			},
			setupDocument: func() (*credit_note_models.CreditNoteModel, error) {
				doc, err := builder.BuildCreditNote()
				if err != nil {
					return nil, err
				}

				err = doc.Summary.SetTotalTaxed(150.0)
				err = doc.Summary.SetTotalExempt(125.0)
				err = doc.Summary.SetTotalNonSubject(10.0)

				return doc, err
			},
			wantErr:   true,
			errorCode: "InvalidCreditNoteTransaction",
		},
		{
			name:        "Exceeds exempt amount",
			branchID:    1,
			originalDTE: "ORIGINAL-DTE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "ORIGINAL-DTE-123").Return(validBalance, nil)
			},
			setupDocument: func() (*credit_note_models.CreditNoteModel, error) {
				doc, err := builder.BuildCreditNote()
				if err != nil {
					return nil, err
				}

				err = doc.Summary.SetTotalTaxed(50.0)
				err = doc.Summary.SetTotalExempt(125.0)
				err = doc.Summary.SetTotalNonSubject(10.0)

				return doc, err
			},
			wantErr:   true,
			errorCode: "InvalidCreditNoteTransaction",
		},
		{
			name:        "Exceeds non-subject amount",
			branchID:    1,
			originalDTE: "ORIGINAL-DTE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "ORIGINAL-DTE-123").Return(validBalance, nil)
			},
			setupDocument: func() (*credit_note_models.CreditNoteModel, error) {
				doc, err := builder.BuildCreditNote()
				if err != nil {
					return nil, err
				}

				err = doc.Summary.SetTotalTaxed(50.0)
				err = doc.Summary.SetTotalExempt(125.0)
				err = doc.Summary.SetTotalNonSubject(110.0)

				return doc, err
			},
			wantErr:   true,
			errorCode: "InvalidCreditNoteTransaction",
		},
		{
			name:        "Failed to get balance control",
			branchID:    1,
			originalDTE: "ORIGINAL-DTE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetDTEBalanceControl(gomock.Any(), uint(1), "ORIGINAL-DTE-123").
					Return(nil, gorm.ErrRecordNotFound)
			},
			setupDocument: func() (*credit_note_models.CreditNoteModel, error) {
				doc, err := builder.BuildCreditNote()
				if err != nil {
					return nil, err
				}

				return doc, nil
			},
			wantErr:   true,
			errorCode: "FailedToGetBalanceControl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document, err := tt.setupDocument()
			if err != nil {
				t.Fatalf("Failed to setup document: %v", err)
			}

			err = nil
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			err = service.ValidateForCreditNote(context.Background(), tt.branchID, tt.originalDTE, document)

			if tt.wantErr {
				assert.Error(t, err)
				test.AssertErrorCode(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDTEServiceUpdateDTE tests the UpdateDTE function
func TestDTEServiceUpdateDTE(t *testing.T) {
	test.TestMain(t)

	dteDetails := dte.DTEDetails{
		ID:            "DTE-123",
		DTEType:       constants.FacturaElectronica,
		ControlNumber: "DTE-01-00000000-000000000000001",
		Status:        constants.DocumentReceived,
	}

	tests := []struct {
		name      string
		branchID  uint
		document  dte.DTEDetails
		setupMock func(*mocks.MockDTERepositoryPort)
		wantErr   bool
		errorCode string
	}{
		{
			name:     "Valid DTE update",
			branchID: 1,
			document: dteDetails,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().Update(gomock.Any(), uint(1), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "Failed to update DTE",
			branchID: 1,
			document: dteDetails,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().Update(gomock.Any(), uint(1), gomock.Any()).
					Return(shared_error.NewFormattedGeneralServiceError("Repository", "Update", "FailedToUpdateDTE"))
			},
			wantErr:   true,
			errorCode: "FailedToUpdateDTE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			err := service.UpdateDTE(context.Background(), tt.branchID, tt.document)

			if tt.wantErr {
				assert.Error(t, err)
				test.AssertErrorCode(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDTEServiceVerifyStatus tests the VerifyStatus function
func TestDTEServiceVerifyStatus(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name       string
		branchID   uint
		id         string
		setupMock  func(*mocks.MockDTERepositoryPort)
		wantStatus string
		wantErr    bool
		errorCode  string
	}{
		{
			name:     "Valid status verification",
			branchID: 1,
			id:       "DTE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().VerifyStatus(gomock.Any(), uint(1), "DTE-123").Return(constants.DocumentReceived, nil)
			},
			wantStatus: constants.DocumentReceived,
			wantErr:    false,
		},
		{
			name:     "Failed to verify status",
			branchID: 1,
			id:       "DTE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().VerifyStatus(gomock.Any(), uint(1), "DTE-123").
					Return("", shared_error.NewFormattedGeneralServiceError("Repository", "VerifyStatus", "FailedToVerifyDTE"))
			},
			wantStatus: "",
			wantErr:    true,
			errorCode:  "FailedToVerifyDTE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			status, err := service.VerifyStatus(context.Background(), tt.branchID, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantStatus, status)
				test.AssertErrorCode(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, status)
			}
		})
	}
}

// TestDTEServiceGetByGenerationCode tests the GetByGenerationCode function
func TestDTEServiceGetByGenerationCode(t *testing.T) {
	test.TestMain(t)

	mockDTE := &dte.DTEDocument{
		Details: &dte.DTEDetails{
			ID:            "GEN-CODE-123",
			DTEType:       constants.FacturaElectronica,
			ControlNumber: "DTE-01-00000000-000000000000001",
			Status:        constants.DocumentReceived,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name           string
		branchID       uint
		generationCode string
		setupMock      func(*mocks.MockDTERepositoryPort)
		wantErr        bool
		errorCode      string
	}{
		{
			name:           "Valid get by generation code",
			branchID:       1,
			generationCode: "GEN-CODE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), "GEN-CODE-123").Return(mockDTE, nil)
			},
			wantErr: false,
		},
		{
			name:           "Failed to get by generation code",
			branchID:       1,
			generationCode: "GEN-CODE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), "GEN-CODE-123").
					Return(nil, shared_error.NewFormattedGeneralServiceError("Repository", "GetByGenerationCode", "FailedToGetDTE"))
			},
			wantErr:   true,
			errorCode: "FailedToGetDTE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			result, err := service.GetByGenerationCode(context.Background(), tt.branchID, tt.generationCode)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				test.AssertErrorCode(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, mockDTE, result)
			}
		})
	}
}

// TestDTEServiceGetByGenerationCodeConsult tests the GetByGenerationCodeConsult function
func TestDTEServiceGetByGenerationCodeConsult(t *testing.T) {
	test.TestMain(t)

	jsonData := `{"receiver":{"name":"Test Customer"},"totals":{"amount":100}}`

	mockDTE := &dte.DTEDocument{
		Details: &dte.DTEDetails{
			ID:            "GEN-CODE-123",
			DTEType:       constants.FacturaElectronica,
			ControlNumber: "DTE-01-00000000-000000000000001",
			Status:        constants.DocumentReceived,
			JSONData:      jsonData,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name           string
		branchID       uint
		generationCode string
		setupMock      func(*mocks.MockDTERepositoryPort)
		wantErr        bool
		errorCode      string
	}{
		{
			name:           "Valid get by generation code consult",
			branchID:       1,
			generationCode: "GEN-CODE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), "GEN-CODE-123").Return(mockDTE, nil)
			},
			wantErr: false,
		},
		{
			name:           "Failed to get by generation code",
			branchID:       1,
			generationCode: "GEN-CODE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), "GEN-CODE-123").
					Return(nil, shared_error.NewFormattedGeneralServiceError("Repository", "GetByGenerationCode", "FailedToGetDTE"))
			},
			wantErr:   true,
			errorCode: "FailedToGetDTE",
		},
		{
			name:           "Invalid JSON data",
			branchID:       1,
			generationCode: "GEN-CODE-123",
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				invalidDTE := *mockDTE
				invalidDTE.Details.JSONData = "{invalid json}"
				mock.EXPECT().GetByGenerationCode(gomock.Any(), uint(1), "GEN-CODE-123").Return(&invalidDTE, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			result, err := service.GetByGenerationCodeConsult(context.Background(), tt.branchID, tt.generationCode)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					test.AssertErrorCode(t, err, tt.errorCode)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, mockDTE.Details.ID, result.GenerationCode)
				assert.Equal(t, mockDTE.Details.ControlNumber, result.ControlNumber)
				assert.Equal(t, mockDTE.Details.Status, result.Status)

				expectedJSON := make(map[string]interface{})
				_ = json.Unmarshal([]byte(jsonData), &expectedJSON)
				assert.Equal(t, expectedJSON, result.JSONData)
			}
		})
	}
}

// TestDTEServiceGetAllDTEs tests the GetAllDTEs function
func TestDTEServiceGetAllDTEs(t *testing.T) {
	test.TestMain(t)

	filters := &dte.DTEFilters{
		Page:     1,
		PageSize: 10,
	}

	summaryWithDocs := &dte.ListSummary{
		Total:    20,
		Received: 15,
		Rejected: 2,
		Pending:  3,
	}

	summaryEmpty := &dte.ListSummary{
		Total:    0,
		Received: 0,
		Rejected: 0,
		Pending:  0,
	}

	documents := []dte.DTEModelResponse{
		{
			TransmissionType: constants.TransmissionContingency,
			Document:         json.RawMessage(`{"id": "DTE-001", "type": "invoice"}`),
			Status:           constants.DocumentReceived,
		},
		{
			TransmissionType: constants.TransmissionContingency,
			Document:         json.RawMessage(`{"id": "DTE-001", "type": "invoice"}`),
			Status:           constants.DocumentReceived,
		},
	}

	tests := []struct {
		name       string
		filters    *dte.DTEFilters
		setupMock  func(*mocks.MockDTERepositoryPort)
		wantErr    bool
		errorCode  string
		checkEmpty bool
	}{
		{
			name:    "Valid get all DTEs with documents",
			filters: filters,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetSummaryStats(gomock.Any(), filters).Return(summaryWithDocs, nil)
				mock.EXPECT().GetPagedDocuments(gomock.Any(), filters).Return(documents, nil)
			},
			wantErr: false,
		},
		{
			name:    "Valid get all DTEs with no documents",
			filters: filters,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetSummaryStats(gomock.Any(), filters).Return(summaryEmpty, nil)
			},
			wantErr:    false,
			checkEmpty: true,
		},
		{
			name:    "Failed to get summary stats",
			filters: filters,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetSummaryStats(gomock.Any(), filters).
					Return(nil, shared_error.NewFormattedGeneralServiceError("Repository", "GetSummaryStats", "FailedToGetSummaryStats"))
			},
			wantErr:   true,
			errorCode: "FailedToGetSummaryStats",
		},
		{
			name:    "Failed to get paged documents",
			filters: filters,
			setupMock: func(mock *mocks.MockDTERepositoryPort) {
				mock.EXPECT().GetSummaryStats(gomock.Any(), filters).Return(summaryWithDocs, nil)
				mock.EXPECT().GetPagedDocuments(gomock.Any(), filters).
					Return(nil, shared_error.NewFormattedGeneralServiceError("Repository", "GetPagedDocuments", "FailedToGetPagedDoc"))
			},
			wantErr:   true,
			errorCode: "FailedToGetPagedDoc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			tt.setupMock(mockRepo)

			service := dte_documents.NewDTEService(mockRepo)

			result, err := service.GetAllDTEs(context.Background(), tt.filters)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				test.AssertErrorCode(t, err, tt.errorCode)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				if tt.checkEmpty {
					assert.Equal(t, int64(0), result.Summary.Total)
					assert.Empty(t, result.Documents)
				} else {
					assert.Equal(t, int64(20), result.Summary.Total)
					assert.Equal(t, 2, len(result.Documents))
				}
			}
		})
	}
}

// TestDTEServiceCreate_RejectsMissingReceptionStamp guards the regression where
// DTEService.Create persisted documents in TransmissionNormal even when the
// reception stamp from Hacienda was nil or empty. After the fix, the service
// must abort with the MissingReceptionStamp i18n code and never reach the
// repository.
func TestDTEServiceCreate_RejectsMissingReceptionStamp(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name           string
		transmission   string
		receptionStamp *string
	}{
		{"nil stamp on normal transmission", constants.TransmissionNormal, nil},
		{"empty stamp on normal transmission", constants.TransmissionNormal, utils.ToStringPointer("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			// expect zero calls to Create — that's the whole point of the regression test
			service := dte_documents.NewDTEService(mockRepo)

			err := service.Create(context.Background(), createMockInvoiceResponse(), tc.transmission, constants.DocumentReceived, tc.receptionStamp)

			assert.Error(t, err)
			test.AssertErrorCode(t, err, "MissingReceptionStamp")
		})
	}
}

// TestDTEServiceCreate_AppendsReceptionStampForAllSupportedTypes guards the
// regression where setReceptionStampIntoAppendix only handled four DTE types,
// silently skipping the appendix injection for nota de débito (06), FSE (14)
// and remisión (04). After the fix, all supported types must successfully run
// through Create and reach the repository.
func TestDTEServiceCreate_AppendsReceptionStampForAllSupportedTypes(t *testing.T) {
	test.TestMain(t)

	supportedTypes := []string{
		constants.FacturaElectronica,
		constants.CCFElectronico,
		constants.NotaCreditoElectronica,
		constants.NotaDebitoElectronica,
		constants.ComprobanteRetencionElectronico,
		constants.NotaRemisionElectronica,
		constants.FacturaSujetoExcluidoElectronica,
	}

	for _, dteType := range supportedTypes {
		t.Run("type_"+dteType, func(t *testing.T) {
			doc := createMockResponseForType(dteType)
			if doc == nil {
				t.Fatalf("no response struct registered for DTE type %s", dteType)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockDTERepositoryPort(ctrl)
			mockRepo.EXPECT().
				Create(gomock.Any(), gomock.Any(), constants.TransmissionNormal, constants.DocumentReceived, gomock.Any()).
				Return(nil)

			service := dte_documents.NewDTEService(mockRepo)
			err := service.Create(context.Background(), doc, constants.TransmissionNormal, constants.DocumentReceived, utils.ToStringPointer("STAMP"))
			assert.NoError(t, err, "Create should accept DTE type %s", dteType)
		})
	}
}
