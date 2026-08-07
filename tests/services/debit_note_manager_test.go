package services

import (
	"context"
	"errors"
	"testing"

	coreDte "github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebitNoteServiceCreate(t *testing.T) {
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
			name: "Valid debit note creation",
			setupData: func() (interface{}, error) {
				debitNote, err := fixtures.BuildValidDebitNote()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsDebitNoteInput(debitNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&coreDte.DTEDocument{
						Details: &coreDte.DTEDetails{
							JSONData: `{"receptor":{"nit":"06140101901011"}}`,
						},
					}, nil).AnyTimes()
				mockDTE.EXPECT().VerifyStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(constants.DocumentReceived, nil).AnyTimes()
				mockDTE.EXPECT().ValidateForDebitNote(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaDebitoElectronica,
					gomock.Any(), gomock.Any(), gomock.Any(),
				).Return("DTE-05-N0010001-000000000012345", nil)
			},
			branchID: 1,
			wantErr:  false,
		},
		{
			name: "Error – no related documents",
			setupData: func() (interface{}, error) {
				debitNote, err := fixtures.BuildValidDebitNote()
				if err != nil {
					return nil, err
				}
				input := fixtures.BuildAsDebitNoteInput(debitNote)
				input.RelatedDocs = nil
				return input, nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {},
			branchID:  1,
			wantErr:   true,
			errorCode: "NoRelatedDocs",
		},
		{
			name: "Error – related document not found",
			setupData: func() (interface{}, error) {
				debitNote, err := fixtures.BuildValidDebitNote()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsDebitNoteInput(debitNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found")).AnyTimes()
			},
			branchID: 1,
			wantErr:  true,
		},
		{
			name: "Error – related document status not received",
			setupData: func() (interface{}, error) {
				debitNote, err := fixtures.BuildValidDebitNote()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsDebitNoteInput(debitNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&coreDte.DTEDocument{
						Details: &coreDte.DTEDetails{
							JSONData: `{"receptor":{"nit":"06140101901011"}}`,
						},
					}, nil).AnyTimes()
				mockDTE.EXPECT().VerifyStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(constants.DocumentRejected, nil).AnyTimes()
			},
			branchID:  1,
			wantErr:   true,
			errorCode: "RelatedDocumentNotReceived",
		},
		{
			name: "Error – sequential number fails",
			setupData: func() (interface{}, error) {
				debitNote, err := fixtures.BuildValidDebitNote()
				if err != nil {
					return nil, err
				}
				return fixtures.BuildAsDebitNoteInput(debitNote), nil
			},
			setupMock: func(mockSeq *mocks.MockSequentialNumberManager, mockDTE *mocks.MockDTEManager) {
				mockDTE.EXPECT().GetByGenerationCode(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&coreDte.DTEDocument{
						Details: &coreDte.DTEDetails{
							JSONData: `{"receptor":{"nit":"06140101901011"}}`,
						},
					}, nil).AnyTimes()
				mockDTE.EXPECT().VerifyStatus(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(constants.DocumentReceived, nil).AnyTimes()
				mockDTE.EXPECT().ValidateForDebitNote(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).AnyTimes()
				mockSeq.EXPECT().GetNextControlNumber(
					gomock.Any(),
					constants.NotaDebitoElectronica,
					gomock.Any(), gomock.Any(), gomock.Any(),
				).Return("", errors.New("seq number error"))
			},
			branchID: 1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
			mockDTE := mocks.NewMockDTEManager(ctrl)
			tt.setupMock(mockSeq, mockDTE)

			svc := debit_note.NewDebitNoteService(mockSeq, mockDTE)

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
