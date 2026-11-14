package application

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/application/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	domainDTE "github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

// TestAdversarialListFiltersByIssuerNIT attacks the all=true scoping rules of the consult use case.
func TestAdversarialListFiltersByIssuerNIT(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name          string
		query         string
		nit           string
		branch        uint
		wantAll       bool
		wantIssuerNIT string
		wantBranchID  uint
	}{
		{"all true with NIT scopes to the issuer", "all=true", "0614-010101-000-0", 7, true, "0614-010101-000-0", 0},
		{"all true with empty NIT falls back to the branch", "all=true", "", 7, false, "", 7},
		{"all absent keeps the branch", "", "0614-010101-000-0", 7, false, "", 7},
		{"all false keeps the branch", "all=false", "0614-010101-000-0", 7, false, "", 7},
		{"all uppercase is not true", "all=TRUE", "0614-010101-000-0", 7, false, "", 7},
		{"all numeric is not true", "all=1", "0614-010101-000-0", 7, false, "", 7},
		{"all with trailing space is not true", "all=true%20", "0614-010101-000-0", 7, false, "", 7},
		{"all repeated uses the first value", "all=false&all=true", "0614-010101-000-0", 7, false, "", 7},
		{"hostile NIT is carried verbatim for binding", "all=true", "' OR 1=1 --", 7, true, "' OR 1=1 --", 0},
		{"client supplied nit param is ignored", "all=true&nit=OTHER&issuer_nit=OTHER&branch_id=99", "0614-010101-000-0", 7, true, "0614-010101-000-0", 0},
		{"client supplied branch param is ignored without all", "branch_id=99&BranchID=99", "0614-010101-000-0", 7, false, "", 7},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			manager := mocks.NewMockDTEManager(ctrl)
			var captured *domainDTE.DTEFilters
			manager.EXPECT().GetAllDTEs(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, filters *domainDTE.DTEFilters) (*domainDTE.DTEListResponse, error) {
					captured = filters
					return &domainDTE.DTEListResponse{}, nil
				})

			request := httptest.NewRequest(http.MethodGet, "/dte?"+tc.query, nil)
			request = request.WithContext(context.WithValue(request.Context(), "claims", &models.AuthClaims{BranchID: tc.branch, NIT: tc.nit}))

			_, err := dte.NewDTEConsultUseCase(manager).GetAllDTEs(request.Context(), request)

			require.NoError(t, err)
			require.NotNil(t, captured)
			assert.Equal(t, tc.wantAll, captured.IncludeAll)
			assert.Equal(t, tc.wantIssuerNIT, captured.IssuerNIT)
			assert.Equal(t, tc.wantBranchID, captured.BranchID)
		})
	}
}
