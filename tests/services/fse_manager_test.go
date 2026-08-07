package services

import (
	"context"
	"errors"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFSEServiceCreate(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name      string
		setupData func() (interface{}, error)
		setupMock func(*mocks.MockSequentialNumberManager)
		branchID  uint
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid FSE creation",
			setupData: func() (interface{}, error) {
				fseDoc, err := fixtures.BuildValidFSE()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsFSEData(fseDoc), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaSujetoExcluidoElectronica,
					gomock.Any(), gomock.Any(), gomock.Any(),
				).Return("DTE-14-N0010001-000000000012345", nil)
			},
			branchID: 1,
			wantErr:  false,
		},
		{
			name: "Error – sequential number fails",
			setupData: func() (interface{}, error) {
				fseDoc, err := fixtures.BuildValidFSE()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsFSEData(fseDoc), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.FacturaSujetoExcluidoElectronica,
					gomock.Any(), gomock.Any(), gomock.Any(),
				).Return("", errors.New("seq number error"))
			},
			branchID: 1,
			wantErr:  true,
		},
		{
			name: "Error – nil items",
			setupData: func() (interface{}, error) {
				fseDoc, err := fixtures.BuildValidFSE()
				if err != nil {
					return nil, err
				}
				data := fixtures.BuildAsFSEData(fseDoc)
				data.Items = nil
				return data, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager) {},
			branchID:  1,
			wantErr:   true,
		},
		{
			name: "Error – nil receiver",
			setupData: func() (interface{}, error) {
				fseDoc, err := fixtures.BuildValidFSE()
				if err != nil {
					return nil, err
				}
				data := fixtures.BuildAsFSEData(fseDoc)
				data.FSEReceiver = nil
				return data, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager) {},
			branchID:  1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
			tt.setupMock(mockSeq)

			svc := fse.NewFSEService(mockSeq)

			data, err := tt.setupData()
			require.NoError(t, err, "failed to set up test data")

			result, err := svc.Create(context.Background(), data, tt.branchID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					assert.Contains(t, err.Error(), tt.errorCode)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
