package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

func TestInvalidationServiceValidate(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name              string
		setupInvalidation func() (*invalidation_models.InvalidationDocument, error)
		wantErr           bool
		errorCode         string
	}{
		{
			name: "Valid Invalidation with Replacement (Type 1)",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				return fixtures.BuildInvalidationWithReplacement()
			},
			wantErr: false,
		},
		{
			name: "Valid Invalidation with Annulment (Type 2)",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				return fixtures.BuildInvalidationWithAnnulment()
			},
			wantErr: false,
		},
		{
			name: "Valid Invalidation Definitive (Type 3)",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				return fixtures.BuildInvalidationDefinitive()
			},
			wantErr: false,
		},
		{
			name: "Error - Invalidation Type 2 with Replacement Code",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				return fixtures.BuildInvalidInvalidation()
			},
			wantErr:   true,
			errorCode: "InvalidField",
		},
		{
			name: "Error - Invalidation Type 1 with Reason",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				return fixtures.BuildInvalidationWithInvalidReason()
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Error - Invalidation Type 3 without Reason",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				return fixtures.BuildInvalidationWithMissingReason()
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Error - Invalidation with Missing Identification",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				doc, _ := fixtures.BuildInvalidationWithReplacement()
				doc.Identification = nil
				return doc, nil
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Error - Invalidation with Missing Issuer",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				doc, _ := fixtures.BuildInvalidationWithReplacement()
				doc.Issuer = nil
				return doc, nil
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Error - Invalidation with Date Out of Range",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				doc, _ := fixtures.BuildInvalidationWithReplacement()

				oldDate, _ := temporal.NewEmissionDate(time.Now().AddDate(0, -6, 0))
				doc.Document.EmissionDate = *oldDate

				return doc, nil
			},
			wantErr:   true,
			errorCode: "InvalidDateForFEFX",
		},
		{
			name: "Error - Invalidation with Invalid Reception Stamp",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				doc, _ := fixtures.BuildInvalidationWithReplacement()
				doc.Document.ReceptionStamp = "INVALID_STAMP"
				return doc, nil
			},
			wantErr:   true,
			errorCode: "InvalidPattern",
		},
		{
			name: "Error - Invalidation with Invalid Document Type",
			setupInvalidation: func() (*invalidation_models.InvalidationDocument, error) {
				doc, _ := fixtures.BuildInvalidationWithReplacement()

				invalidType := document.NewValidatedDTEType("99")
				doc.Document.Type = *invalidType

				return doc, nil
			},
			wantErr:   true,
			errorCode: "InvalidDTETypeForInvalidation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalidationDoc, err := tt.setupInvalidation()
			if err != nil {
				t.Fatalf("Error preparing test data: %v", err)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDTEManager := mocks.NewMockDTEManager(ctrl)
			service := invalidation.NewInvalidationService(mockDTEManager)

			err = service.Validate(context.Background(), 1, invalidationDoc)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					var dteErr *dte_errors.DTEError
					if errors.As(err, &dteErr) {
						assert.Contains(t, dteErr.Error(), tt.errorCode, "Error message should contain expected code")
					} else {
						assert.Contains(t, err.Error(), tt.errorCode, "Error message should contain expected code")
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInvalidationServiceValidateStatus(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name         string
		setupRequest func() structs.CreateInvalidationRequest
		setupMock    func(*mocks.MockDTEManager)
		wantErr      bool
		errorCode    string
	}{
		{
			name: "Valid Status - Type 1 Invalidation",
			setupRequest: func() structs.CreateInvalidationRequest {
				replacementCode := "DTE-01-00000001-000000000000002"
				return structs.CreateInvalidationRequest{
					GenerationCode:            "DTE-01-00000001-000000000000001",
					ReplacementGenerationCode: &replacementCode,
					Reason: &structs.ReasonRequest{
						Type: 1,
					},
				}
			},
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000001",
				).Return(constants.DocumentReceived, nil)

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000002",
				).Return(constants.DocumentReceived, nil)
			},
			wantErr: false,
		},
		{
			name: "Valid Status - Type 2 Invalidation",
			setupRequest: func() structs.CreateInvalidationRequest {
				return structs.CreateInvalidationRequest{
					GenerationCode: "DTE-01-00000001-000000000000001",
					Reason: &structs.ReasonRequest{
						Type: 2,
					},
				}
			},
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000001",
				).Return(constants.DocumentReceived, nil)
			},
			wantErr: false,
		},
		{
			name: "Error - Original Document Already Invalid",
			setupRequest: func() structs.CreateInvalidationRequest {
				return structs.CreateInvalidationRequest{
					GenerationCode: "DTE-01-00000001-000000000000001",
					Reason: &structs.ReasonRequest{
						Type: 2,
					},
				}
			},
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000001",
				).Return(constants.DocumentInvalid, nil)
			},
			wantErr:   true,
			errorCode: "DocumentAlreadyInvalid",
		},
		{
			name: "Error - Original Document Rejected",
			setupRequest: func() structs.CreateInvalidationRequest {
				return structs.CreateInvalidationRequest{
					GenerationCode: "DTE-01-00000001-000000000000001",
					Reason: &structs.ReasonRequest{
						Type: 2,
					},
				}
			},
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000001",
				).Return(constants.DocumentRejected, nil)
			},
			wantErr:   true,
			errorCode: "DocumentReject",
		},
		{
			name: "Error - Original Document Pending",
			setupRequest: func() structs.CreateInvalidationRequest {
				return structs.CreateInvalidationRequest{
					GenerationCode: "DTE-01-00000001-000000000000001",
					Reason: &structs.ReasonRequest{
						Type: 2,
					},
				}
			},
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000001",
				).Return(constants.DocumentPending, nil)
			},
			wantErr:   true,
			errorCode: "DocumentPending",
		},
		{
			name: "Error - Replacement Document Invalid",
			setupRequest: func() structs.CreateInvalidationRequest {
				replacementCode := "DTE-01-00000001-000000000000002"
				return structs.CreateInvalidationRequest{
					GenerationCode:            "DTE-01-00000001-000000000000001",
					ReplacementGenerationCode: &replacementCode,
					Reason: &structs.ReasonRequest{
						Type: 1,
					},
				}
			},
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000001",
				).Return(constants.DocumentReceived, nil)

				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000002",
				).Return(constants.DocumentInvalid, nil)
			},
			wantErr:   true,
			errorCode: "DocumentAlreadyInvalid",
		},
		{
			name: "Error - Document Not Found",
			setupRequest: func() structs.CreateInvalidationRequest {
				return structs.CreateInvalidationRequest{
					GenerationCode: "DTE-01-00000001-000000000000001",
					Reason: &structs.ReasonRequest{
						Type: 2,
					},
				}
			},
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().VerifyStatus(
					gomock.Any(),
					gomock.Any(),
					"DTE-01-00000001-000000000000001",
				).Return("", shared_error.NewGeneralServiceError("DTEManager", "VerifyStatus", "Document not found", nil))
			},
			wantErr:   true,
			errorCode: "Document not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.setupRequest()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDTEManager := mocks.NewMockDTEManager(ctrl)
			tt.setupMock(mockDTEManager)

			service := invalidation.NewInvalidationService(mockDTEManager)

			err := service.ValidateStatus(context.Background(), 1, request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					var serviceErr *shared_error.ServiceError
					if errors.As(err, &serviceErr) {
						assert.Contains(t, serviceErr.Error(), tt.errorCode, "Error message should contain expected code")
					} else {
						assert.Contains(t, err.Error(), tt.errorCode, "Error message should contain expected code")
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInvalidationServiceInvalidateDocument(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name      string
		setupMock func(*mocks.MockDTEManager)
		wantErr   bool
		errorCode string
	}{
		{
			name: "Successful Document Invalidation",
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().UpdateDTE(
					gomock.Any(),
					gomock.Any(),
					dte.DTEDetails{
						ID:     "DTE-01-00000001-000000000000001",
						Status: constants.DocumentInvalid,
					},
				).Return(nil)

				mockDTE.EXPECT().GetByGenerationCode(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dte.DTEDocument{
					Details: &dte.DTEDetails{
						ID:      "DTE-01-00000001-000000000000001",
						DTEType: constants.CCFElectronico,
					},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "Error During Document Invalidation",
			setupMock: func(mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().UpdateDTE(
					gomock.Any(),
					gomock.Any(),
					dte.DTEDetails{
						ID:     "DTE-01-00000001-000000000000001",
						Status: constants.DocumentInvalid,
					},
				).Return(errors.New("any error"))
			},
			wantErr:   true,
			errorCode: "FailedToInvalidatedDTE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDTEManager := mocks.NewMockDTEManager(ctrl)
			tt.setupMock(mockDTEManager)

			service := invalidation.NewInvalidationService(mockDTEManager)

			err := service.InvalidateDocument(context.Background(), 1, "DTE-01-00000001-000000000000001")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					assert.Contains(t, err.Error(), tt.errorCode, "Error message should contain expected code")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
