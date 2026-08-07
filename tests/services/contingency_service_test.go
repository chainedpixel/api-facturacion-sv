package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	coreDte "github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	coreUser "github.com/chainedpixel/ordo-factus/internal/domain/core/user"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// buildContingencyCtx returns a context with valid AuthClaims.
func buildContingencyCtx(branchID uint) context.Context {
	return context.WithValue(context.Background(), "claims", &authModels.AuthClaims{
		ClientID: 1,
		BranchID: branchID,
		NIT:      "0614-010101-000-0",
	})
}

// minimalDTEJSON returns the smallest valid DTE JSON understood by ExtractAuxiliarIdentification.
func minimalDTEJSON() string {
	return `{
		"identificacion": {
			"tipoDte":          "01",
			"numeroControl":    "DTE-01-00000001-000000000000001",
			"codigoGeneracion": "AB123456-1234-1234-1234-AB1234567890"
		},
		"emisor": {
			"nit": "0614-010101-000-0"
		}
	}`
}

// TestContingencyService_StoreDocumentInContingency covers happy-path and error branches
// for StoreDocumentInContingency.
func TestContingencyService_StoreDocumentInContingency(t *testing.T) {
	test.TestMain(t)

	type args struct {
		ctx             context.Context
		document        interface{}
		dteType         string
		contingencyType int8
		reason          string
	}

	type mockSetup struct {
		dteManager        func(*mocks.MockDTEManager)
		repo              func(*mocks.MockContingencyRepositoryPort)
		sequentialManager func(*mocks.MockSequentialNumberManager)
	}

	validDoc := json.RawMessage(minimalDTEJSON())

	cases := []struct {
		name      string
		args      args
		setup     mockSetup
		wantErr   bool
		errSubstr string
	}{
		{
			name: "success – document stored in contingency",
			args: args{
				ctx:             buildContingencyCtx(1),
				document:        validDoc,
				dteType:         constants.FacturaElectronica,
				contingencyType: constants.FallaServicioInternet,
				reason:          "Internet outage",
			},
			setup: mockSetup{
				dteManager: func(m *mocks.MockDTEManager) {
					m.EXPECT().
						Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, nil).
						Return(nil)
				},
				repo: func(m *mocks.MockContingencyRepositoryPort) {
					m.EXPECT().
						Create(gomock.Any(), gomock.Any()).
						Return(nil)
				},
				sequentialManager: func(m *mocks.MockSequentialNumberManager) {
					m.EXPECT().
						MarkReservationAsContingencyByControlNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil)
				},
			},
			wantErr: false,
		},
		{
			name: "error – DTE manager Create fails",
			args: args{
				ctx:             buildContingencyCtx(1),
				document:        validDoc,
				dteType:         constants.FacturaElectronica,
				contingencyType: constants.FallaEnergiaElectrica,
				reason:          "Power failure",
			},
			setup: mockSetup{
				dteManager: func(m *mocks.MockDTEManager) {
					m.EXPECT().
						Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, nil).
						Return(errors.New("db error"))
				},
				repo:              func(m *mocks.MockContingencyRepositoryPort) {},
				sequentialManager: func(m *mocks.MockSequentialNumberManager) {},
			},
			wantErr:   true,
			errSubstr: "StoreDocumentInContingency",
		},
		{
			name: "error – repository Create fails",
			args: args{
				ctx:             buildContingencyCtx(2),
				document:        validDoc,
				dteType:         constants.FacturaElectronica,
				contingencyType: constants.NoDisponibilidadMH,
				reason:          "MH unavailable",
			},
			setup: mockSetup{
				dteManager: func(m *mocks.MockDTEManager) {
					m.EXPECT().
						Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, nil).
						Return(nil)
				},
				repo: func(m *mocks.MockContingencyRepositoryPort) {
					m.EXPECT().
						Create(gomock.Any(), gomock.Any()).
						Return(errors.New("repo error"))
				},
				sequentialManager: func(m *mocks.MockSequentialNumberManager) {},
			},
			wantErr:   true,
			errSubstr: "StoreDocumentInContingency",
		},
		{
			name: "error – empty GenerationCode is rejected before persisting",
			args: args{
				ctx: buildContingencyCtx(1),
				document: json.RawMessage(`{
					"identificacion": {
						"tipoDte":          "01",
						"numeroControl":    "DTE-01-00000001-000000000000001",
						"codigoGeneracion": ""
					},
					"emisor": {"nit": "0614-010101-000-0"}
				}`),
				dteType:         constants.FacturaElectronica,
				contingencyType: constants.OtroMotivo,
				reason:          "Other",
			},
			setup: mockSetup{
				dteManager:        func(m *mocks.MockDTEManager) {},
				repo:              func(m *mocks.MockContingencyRepositoryPort) {},
				sequentialManager: func(m *mocks.MockSequentialNumberManager) {},
			},
			wantErr:   true,
			errSubstr: "MissingDTEIdentification",
		},
		{
			name: "error – empty ControlNumber is rejected before persisting",
			args: args{
				ctx: buildContingencyCtx(1),
				document: json.RawMessage(`{
					"identificacion": {
						"tipoDte":          "01",
						"numeroControl":    "",
						"codigoGeneracion": "AB123456-1234-1234-1234-AB1234567890"
					},
					"emisor": {"nit": "0614-010101-000-0"}
				}`),
				dteType:         constants.FacturaElectronica,
				contingencyType: constants.OtroMotivo,
				reason:          "Other",
			},
			setup: mockSetup{
				dteManager:        func(m *mocks.MockDTEManager) {},
				repo:              func(m *mocks.MockContingencyRepositoryPort) {},
				sequentialManager: func(m *mocks.MockSequentialNumberManager) {},
			},
			wantErr:   true,
			errSubstr: "MissingDTEIdentification",
		},
		{
			name: "warning only – MarkReservationAsContingencyByControlNumber fails (non-fatal)",
			args: args{
				ctx:             buildContingencyCtx(1),
				document:        validDoc,
				dteType:         constants.FacturaElectronica,
				contingencyType: constants.OtroMotivo,
				reason:          "Other",
			},
			setup: mockSetup{
				dteManager: func(m *mocks.MockDTEManager) {
					m.EXPECT().
						Create(gomock.Any(), gomock.Any(), constants.TransmissionContingency, constants.DocumentPending, nil).
						Return(nil)
				},
				repo: func(m *mocks.MockContingencyRepositoryPort) {
					m.EXPECT().
						Create(gomock.Any(), gomock.Any()).
						Return(nil)
				},
				sequentialManager: func(m *mocks.MockSequentialNumberManager) {
					m.EXPECT().
						MarkReservationAsContingencyByControlNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(errors.New("reservation not found"))
				},
			},
			wantErr: false, // warning only, not a fatal error
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDTE := mocks.NewMockDTEManager(ctrl)
			mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
			mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
			mockEvent := mocks.NewMockContingencyEventSender(ctrl)

			tc.setup.dteManager(mockDTE)
			tc.setup.repo(mockRepo)
			tc.setup.sequentialManager(mockSeq)

			svc := contingency.NewContingencyManager(
				nil, mockDTE, mockRepo,
				nil, nil, nil, nil, nil,
				mockEvent, mockSeq, nil, nil,
			)

			err := svc.StoreDocumentInContingency(
				tc.args.ctx,
				tc.args.document,
				tc.args.dteType,
				tc.args.contingencyType,
				tc.args.reason,
			)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// buildPendingDoc builds a ContingencyDocument with the given NIT and DTE type.
func buildPendingDoc(nit, dteType string) coreDte.ContingencyDocument {
	return coreDte.ContingencyDocument{
		ID:              "contingency-1",
		DocumentID:      "AABB1122-0000-0000-0000-AABB11220000",
		BranchID:        1,
		ContingencyType: int8(constants.FallaServicioInternet),
		Reason:          "Test reason",
		Document: &coreDte.DTEDetails{
			ID:            "AABB1122-0000-0000-0000-AABB11220000",
			DTEType:       dteType,
			ControlNumber: "DTE-01-00000001-000000000000001",
			JSONData:      `{"identificacion":{"tipoDte":"01","numeroControl":"DTE-01-00000001-000000000000001","codigoGeneracion":"AABB1122-0000-0000-0000-AABB11220000"},"emisor":{"nit":"` + nit + `"}}`,
		},
		Branch: &coreUser.BranchOffice{
			User: &coreUser.User{NIT: nit},
		},
	}
}

// TestContingencyService_RetransmitPendingDocuments covers the main retransmit flow.
func TestContingencyService_RetransmitPendingDocuments(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name    string
		setup   func(*mocks.MockContingencyRepositoryPort, *mocks.MockContingencyEventSender)
		wantErr bool
	}{
		{
			name: "no pending documents – returns nil immediately",
			setup: func(repo *mocks.MockContingencyRepositoryPort, event *mocks.MockContingencyEventSender) {
				repo.EXPECT().
					GetPending(gomock.Any(), gomock.Any()).
					Return([]coreDte.ContingencyDocument{}, nil)
			},
			wantErr: false,
		},
		{
			name: "error fetching pending documents",
			setup: func(repo *mocks.MockContingencyRepositoryPort, event *mocks.MockContingencyEventSender) {
				repo.EXPECT().
					GetPending(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name: "pending documents – event send fails (non-ContingencyEventExistsError) – continues",
			setup: func(repo *mocks.MockContingencyRepositoryPort, event *mocks.MockContingencyEventSender) {
				docs := []coreDte.ContingencyDocument{buildPendingDoc("0614-010101-000-0", "01")}
				repo.EXPECT().
					GetPending(gomock.Any(), gomock.Any()).
					Return(docs, nil)
				event.EXPECT().
					PrepareAndSendContingencyEvent(gomock.Any(), gomock.Any()).
					Return(errors.New("event send failure"))
			},
			wantErr: false, // continues to next NIT group, returns nil
		},
		{
			name: "two NITs – PrepareAndSendContingencyEvent called once per NIT group",
			setup: func(repo *mocks.MockContingencyRepositoryPort, event *mocks.MockContingencyEventSender) {
				docs := []coreDte.ContingencyDocument{
					buildPendingDoc("NIT-A", "01"),
					buildPendingDoc("NIT-A", "01"),
					buildPendingDoc("NIT-B", "03"),
				}
				repo.EXPECT().
					GetPending(gomock.Any(), gomock.Any()).
					Return(docs, nil)
				// Each NIT group triggers one PrepareAndSendContingencyEvent call.
				// We return an error so the code skips processSystemDocumentsByType
				// (which requires additional mocks). The important invariant is that
				// the event sender is called exactly once per NIT group (2 groups).
				event.EXPECT().
					PrepareAndSendContingencyEvent(gomock.Any(), gomock.Any()).
					Return(errors.New("simulated send failure")).
					Times(2)
			},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
			mockEvent := mocks.NewMockContingencyEventSender(ctrl)
			mockDTE := mocks.NewMockDTEManager(ctrl)
			mockSeq := mocks.NewMockSequentialNumberManager(ctrl)

			tc.setup(mockRepo, mockEvent)

			svc := contingency.NewContingencyManager(
				nil, mockDTE, mockRepo,
				nil, nil, nil, nil, nil,
				mockEvent, mockSeq, nil, nil,
			)

			err := svc.RetransmitPendingDocuments(context.Background())

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestContingencyEventExistsError tests the error type itself.
func TestContingencyEventExistsError(t *testing.T) {
	docs := []coreDte.ContingencyDocument{{ID: "doc-1"}}
	err := &contingency.ContingencyEventExistsError{
		Message:      "already exists",
		Documents:    docs,
		Observations: "some observations",
	}

	assert.Equal(t, "contingency event already exists: already exists", err.Error())
	assert.Len(t, err.Documents, 1)
	assert.Equal(t, "doc-1", err.Documents[0].ID)
}

// TestContingencyService_RetransmitPendingDocuments_EventExistsPath covers the
// ContingencyEventExistsError branch: the event sender reports the event already
// exists, verifyAndUpdateExistingDocuments runs (with empty docs → no-op), then
// processSystemDocumentsByType is called but getTokenAndCreds fails immediately
// because the auth manager returns an error.  The outer function must still return nil.
func TestContingencyService_RetransmitPendingDocuments_EventExistsPath(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockContingencyRepositoryPort(ctrl)
	mockEvent := mocks.NewMockContingencyEventSender(ctrl)
	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockSeq := mocks.NewMockSequentialNumberManager(ctrl)
	mockAuth := mocks.NewMockAuthManager(ctrl)

	// One pending document for NIT-A, type 01.
	docs := []coreDte.ContingencyDocument{buildPendingDoc("0614-010101-000-0", "01")}
	mockRepo.EXPECT().
		GetPending(gomock.Any(), gomock.Any()).
		Return(docs, nil)

	// Event sender returns ContingencyEventExistsError with empty Documents
	// so verifyAndUpdateExistingDocuments is a no-op loop.
	mockEvent.EXPECT().
		PrepareAndSendContingencyEvent(gomock.Any(), gomock.Any()).
		Return(&contingency.ContingencyEventExistsError{
			Message:   "event already exists",
			Documents: nil,
		})

	// After the ContingencyEventExistsError path, processSystemDocumentsByType is
	// invoked.  getTokenAndCreds calls authManager.GetBranchByBranchID.  We return
	// an error so the function logs it and continues without panicking.
	mockAuth.EXPECT().
		GetBranchByBranchID(gomock.Any(), uint(1)).
		Return(nil, errors.New("branch not found in test"))

	svc := contingency.NewContingencyManager(
		mockAuth, mockDTE, mockRepo,
		nil, nil, nil, nil, nil,
		mockEvent, mockSeq, nil, nil,
	)

	err := svc.RetransmitPendingDocuments(context.Background())
	assert.NoError(t, err, "ContingencyEventExistsError path must not propagate error to caller")
}
