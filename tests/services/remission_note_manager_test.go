package services

import (
	"context"
	"errors"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemissionNoteServiceCreate(t *testing.T) {
	test.TestMain(t)

	tests := []struct {
		name      string
		setupData func() (interface{}, error)
		setupMock func(*mocks.MockSequentialNumberManager, *mocks.MockDTEManager)
		branchID  uint
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid remission note creation",
			setupData: func() (interface{}, error) {
				remissionNoteDoc, receiver, err := fixtures.BuildValidRemissionNote()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsRemissionNoteInput(remissionNoteDoc, receiver), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaRemisionElectronica,
					gomock.Any(), gomock.Any(), gomock.Any(),
				).Return("DTE-04-N0010001-000000000012345", nil)
			},
			branchID: 1,
			wantErr:  false,
		},
		{
			name: "Error – sequential number fails",
			setupData: func() (interface{}, error) {
				remissionNoteDoc, receiver, err := fixtures.BuildValidRemissionNote()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsRemissionNoteInput(remissionNoteDoc, receiver), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaRemisionElectronica,
					gomock.Any(), gomock.Any(), gomock.Any(),
				).Return("", errors.New("seq number error"))
			},
			branchID: 1,
			wantErr:  true,
		},
		{
			name: "Error – nil items",
			setupData: func() (interface{}, error) {
				remissionNoteDoc, receiver, err := fixtures.BuildValidRemissionNote()
				if err != nil {
					return nil, err
				}
				input := fixtures.BuildAsRemissionNoteInput(remissionNoteDoc, receiver)
				input.Items = nil
				return input, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {},
			branchID:  1,
			wantErr:   true,
		},
		{
			name: "Error – nil receiver",
			setupData: func() (interface{}, error) {
				remissionNoteDoc, receiver, err := fixtures.BuildValidRemissionNote()
				if err != nil {
					return nil, err
				}
				input := fixtures.BuildAsRemissionNoteInput(remissionNoteDoc, receiver)
				input.Receiver = nil
				return input, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {},
			branchID:  1,
			wantErr:   true,
		},
		{
			name: "Error – nil summary",
			setupData: func() (interface{}, error) {
				remissionNoteDoc, receiver, err := fixtures.BuildValidRemissionNote()
				if err != nil {
					return nil, err
				}
				input := fixtures.BuildAsRemissionNoteInput(remissionNoteDoc, receiver)
				input.RemissionSummary = nil
				return input, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {},
			branchID:  1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
			mockDTE := mocks.NewMockDTEManager(ctrl)
			tt.setupMock(mockSeq, mockDTE)

			svc := remission_note.NewRemissionNoteService(mockSeq, mockDTE)

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
