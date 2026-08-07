package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSequentialNumberServiceGetNextControlNumber(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name              string
		dteType           string
		branchID          uint
		posCode           *string
		establishmentCode *string
		setupMocks        func(*mocks.MockSequentialNumberRepositoryPort, *mocks.MockAuthRepositoryPort, *mocks.MockReservedSequenceRepositoryPort)
		expectedFormat    string
		wantErr           bool
		errorMsg          string
	}{
		{
			name:              "Valid with year in DTE",
			dteType:           constants.FacturaElectronica,
			branchID:          1,
			posCode:           utils.ToStringPointer("0001"),
			establishmentCode: utils.ToStringPointer("C002"),
			setupMocks: func(seqMock *mocks.MockSequentialNumberRepositoryPort, authMock *mocks.MockAuthRepositoryPort, reservedMock *mocks.MockReservedSequenceRepositoryPort) {
				user := &user.User{YearInDTE: true}
				authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(user, nil)
				reservedMock.EXPECT().GetOldestReleasedNumber(gomock.Any(), uint(1), constants.FacturaElectronica, gomock.Any()).Return(nil, fmt.Errorf("not found"))
				seqMock.EXPECT().GetNext(gomock.Any(), constants.FacturaElectronica, uint(1)).Return(12345, nil)
				reservedMock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(uint(1), nil)
			},
			expectedFormat: "DTE-01-C0020001-YYYY00000012345",
			wantErr:        false,
		},
		{
			name:              "Valid without year in DTE",
			dteType:           constants.FacturaElectronica,
			branchID:          1,
			posCode:           utils.ToStringPointer("0001"),
			establishmentCode: utils.ToStringPointer("C002"),
			setupMocks: func(seqMock *mocks.MockSequentialNumberRepositoryPort, authMock *mocks.MockAuthRepositoryPort, reservedMock *mocks.MockReservedSequenceRepositoryPort) {
				user := &user.User{YearInDTE: false}
				authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(user, nil)
				reservedMock.EXPECT().GetOldestReleasedNumber(gomock.Any(), uint(1), constants.FacturaElectronica, gomock.Any()).Return(nil, fmt.Errorf("not found"))
				seqMock.EXPECT().GetNext(gomock.Any(), constants.FacturaElectronica, uint(1)).Return(12345, nil)
				reservedMock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(uint(1), nil)
			},
			expectedFormat: "DTE-01-C0020001-000000000012345",
			wantErr:        false,
		},
		{
			name:              "Valid CCF with default codes",
			dteType:           constants.CCFElectronico,
			branchID:          1,
			posCode:           nil,
			establishmentCode: nil,
			setupMocks: func(seqMock *mocks.MockSequentialNumberRepositoryPort, authMock *mocks.MockAuthRepositoryPort, reservedMock *mocks.MockReservedSequenceRepositoryPort) {
				user := &user.User{YearInDTE: false}
				authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(user, nil)
				reservedMock.EXPECT().GetOldestReleasedNumber(gomock.Any(), uint(1), constants.CCFElectronico, gomock.Any()).Return(nil, fmt.Errorf("not found"))
				seqMock.EXPECT().GetNext(gomock.Any(), constants.CCFElectronico, uint(1)).Return(12345, nil)
				reservedMock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(uint(1), nil)
			},
			expectedFormat: "DTE-03-00000000-000000000012345",
			wantErr:        false,
		},
		{
			name:              "Valid credit note",
			dteType:           constants.NotaCreditoElectronica,
			branchID:          1,
			posCode:           utils.ToStringPointer("0003"),
			establishmentCode: utils.ToStringPointer("C004"),
			setupMocks: func(seqMock *mocks.MockSequentialNumberRepositoryPort, authMock *mocks.MockAuthRepositoryPort, reservedMock *mocks.MockReservedSequenceRepositoryPort) {
				user := &user.User{YearInDTE: true}
				authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(user, nil)
				reservedMock.EXPECT().GetOldestReleasedNumber(gomock.Any(), uint(1), constants.NotaCreditoElectronica, gomock.Any()).Return(nil, fmt.Errorf("not found"))
				seqMock.EXPECT().GetNext(gomock.Any(), constants.NotaCreditoElectronica, uint(1)).Return(67890, nil)
				reservedMock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(uint(1), nil)
			},
			expectedFormat: "DTE-05-C0040003-YYYY00000067890",
			wantErr:        false,
		},
		{
			name:              "Fails to get user",
			dteType:           constants.FacturaElectronica,
			branchID:          1,
			posCode:           utils.ToStringPointer("0001"),
			establishmentCode: utils.ToStringPointer("C002"),
			setupMocks: func(seqMock *mocks.MockSequentialNumberRepositoryPort, authMock *mocks.MockAuthRepositoryPort, reservedMock *mocks.MockReservedSequenceRepositoryPort) {
				authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(nil,
					shared_error.NewGeneralServiceError("AuthRepository", "GetByBranchID", "User not found", nil))
			},
			wantErr:  true,
			errorMsg: "failed to get user by branchID",
		},
		{
			name:              "Fails to get next sequential number",
			dteType:           constants.FacturaElectronica,
			branchID:          1,
			posCode:           utils.ToStringPointer("0001"),
			establishmentCode: utils.ToStringPointer("C002"),
			setupMocks: func(seqMock *mocks.MockSequentialNumberRepositoryPort, authMock *mocks.MockAuthRepositoryPort, reservedMock *mocks.MockReservedSequenceRepositoryPort) {
				user := &user.User{YearInDTE: true}
				authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(user, nil)
				reservedMock.EXPECT().GetOldestReleasedNumber(gomock.Any(), uint(1), constants.FacturaElectronica, gomock.Any()).Return(nil, fmt.Errorf("not found"))
				seqMock.EXPECT().GetNext(gomock.Any(), constants.FacturaElectronica, uint(1)).Return(0,
					shared_error.NewGeneralServiceError("SequentialNumberRepository", "GetNext", "Failed to get next number", nil))
			},
			wantErr:  true,
			errorMsg: "failed to get next control number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			seqMock := mocks.NewMockSequentialNumberRepositoryPort(ctrl)
			authMock := mocks.NewMockAuthRepositoryPort(ctrl)
			reservedMock := mocks.NewMockReservedSequenceRepositoryPort(ctrl)

			tt.setupMocks(seqMock, authMock, reservedMock)

			service := dte_documents.NewSequentialNumberService(seqMock, authMock, reservedMock)

			controlNumber, err := service.GetNextControlNumber(context.Background(), tt.dteType, tt.branchID, tt.posCode, tt.establishmentCode)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "error message doesn't match expected content")
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, controlNumber)
				assert.Equal(t, 31, len(controlNumber), "length of control number should be 31 characters")

				assert.True(t, strings.HasPrefix(controlNumber, fmt.Sprintf("DTE-%s-", tt.dteType)))

				if tt.expectedFormat != "" {
					parts := strings.Split(controlNumber, "-")
					expectedParts := strings.Split(tt.expectedFormat, "-")

					assert.Equal(t, len(expectedParts), len(parts))
					assert.Equal(t, expectedParts[0], parts[0])
					assert.Equal(t, expectedParts[1], parts[1])
					assert.Equal(t, len(expectedParts[2]), len(parts[2]))

					if strings.Contains(tt.expectedFormat, "YYYY") {
						assert.Equal(t, 15, len(parts[3]))
						yearStr := parts[3][:4]
						_, err := time.Parse("2006", yearStr)
						assert.NoError(t, err)
						seqPart := parts[3][4:]
						assert.Equal(t, 11, len(seqPart))
					} else {
						assert.Equal(t, 15, len(parts[3]))
					}
				}
			}
		})
	}
}

func TestSequentialNumberServiceWithDifferentDTETypes(t *testing.T) {
	test.TestMain(t)

	dteTypes := []string{
		constants.FacturaElectronica,
		constants.CCFElectronico,
		constants.NotaCreditoElectronica,
		constants.NotaRemisionElectronica,
		constants.ComprobanteRetencionElectronico,
	}

	for _, dteType := range dteTypes {
		t.Run(fmt.Sprintf("DTE type: %s", dteType), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			seqMock := mocks.NewMockSequentialNumberRepositoryPort(ctrl)
			authMock := mocks.NewMockAuthRepositoryPort(ctrl)
			reservedMock := mocks.NewMockReservedSequenceRepositoryPort(ctrl)

			user := &user.User{YearInDTE: false}

			authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(user, nil)
			reservedMock.EXPECT().GetOldestReleasedNumber(gomock.Any(), uint(1), dteType, gomock.Any()).Return(nil, fmt.Errorf("not found"))
			seqMock.EXPECT().GetNext(gomock.Any(), dteType, uint(1)).Return(12345, nil)
			reservedMock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(uint(1), nil)

			service := dte_documents.NewSequentialNumberService(seqMock, authMock, reservedMock)

			controlNumber, err := service.GetNextControlNumber(context.Background(), dteType, 1, nil, nil)

			assert.NoError(t, err)
			assert.NotEmpty(t, controlNumber)

			expectedPrefix := fmt.Sprintf("DTE-%s-", dteType)
			assert.True(t, strings.HasPrefix(controlNumber, expectedPrefix))

			assert.Equal(t, 31, len(controlNumber), "length of control number should be 31 characters")
		})
	}
}

func TestSequentialNumberServiceWithVariousConfigurations(t *testing.T) {
	test.TestMain(t)

	configurations := []struct {
		name              string
		posCode           *string
		establishmentCode *string
		yearInDTE         bool
	}{
		{"Default values", nil, nil, false},
		{"Only POS code", utils.ToStringPointer("0123"), nil, false},
		{"Only establishment code", nil, utils.ToStringPointer("C456"), false},
		{"Both codes with year", utils.ToStringPointer("0123"), utils.ToStringPointer("C456"), true},
	}

	for _, config := range configurations {
		t.Run(config.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			seqMock := mocks.NewMockSequentialNumberRepositoryPort(ctrl)
			authMock := mocks.NewMockAuthRepositoryPort(ctrl)
			reservedMock := mocks.NewMockReservedSequenceRepositoryPort(ctrl)

			user := &user.User{YearInDTE: config.yearInDTE}

			authMock.EXPECT().GetByBranchID(gomock.Any(), uint(1)).Return(user, nil)
			reservedMock.EXPECT().GetOldestReleasedNumber(gomock.Any(), uint(1), constants.FacturaElectronica, gomock.Any()).Return(nil, fmt.Errorf("not found"))
			seqMock.EXPECT().GetNext(gomock.Any(), constants.FacturaElectronica, uint(1)).Return(12345, nil)
			reservedMock.EXPECT().Create(gomock.Any(), gomock.Any()).Return(uint(1), nil)

			service := dte_documents.NewSequentialNumberService(seqMock, authMock, reservedMock)

			controlNumber, err := service.GetNextControlNumber(
				context.Background(),
				constants.FacturaElectronica,
				1,
				config.posCode,
				config.establishmentCode,
			)

			assert.NoError(t, err)
			assert.NotEmpty(t, controlNumber)
			assert.Equal(t, 31, len(controlNumber), "length of control number should be 31 characters")

			parts := strings.Split(controlNumber, "-")
			assert.Equal(t, 4, len(parts))

			lastPart := parts[3]
			assert.Equal(t, 15, len(lastPart))

			if config.yearInDTE {
				yearStr := lastPart[:4]
				_, err := time.Parse("2006", yearStr)
				assert.NoError(t, err)
			} else {
				assert.True(t, strings.HasPrefix(lastPart, "000000"))
			}
		})
	}
}
