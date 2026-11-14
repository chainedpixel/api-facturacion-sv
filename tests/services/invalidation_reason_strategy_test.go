package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/calendar"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/validator/strategy"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
)

// TestInvalidationReasonStrategy_ReplacementCode covers when the replacement code is required or forbidden.
func TestInvalidationReasonStrategy_ReplacementCode(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name            string
		dteType         string
		invalidation    int
		withReplacement bool
		wantCode        string
	}{
		{"credit note type 1 without replacement", constants.NotaCreditoElectronica, 1, false, ""},
		{"credit note type 1 with replacement", constants.NotaCreditoElectronica, 1, true, "ReplacementCodeNotAllowedForType"},
		{"credit note type 3 without replacement", constants.NotaCreditoElectronica, 3, false, ""},
		{"invoice type 1 without replacement", constants.FacturaElectronica, 1, false, "InvalidReplacementCode1And3"},
		{"invoice type 1 with replacement", constants.FacturaElectronica, 1, true, ""},
		{"invoice type 2 with replacement", constants.FacturaElectronica, 2, true, "InvalidInvalidationType2"},
		{"invoice type 2 without replacement", constants.FacturaElectronica, 2, false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := fixtures.NewInvalidationBuilder().
				AddIdentification().
				AddIssuer().
				AddInvalidatedDocument().
				AddInvalidationReason(tc.invalidation).
				BuildWithoutValidation()
			assert.NoError(t, err)

			docType, err := document.NewDTEType(tc.dteType)
			assert.NoError(t, err)
			doc.Document.Type = *docType
			doc.Document.ReplacementCode = nil
			if tc.withReplacement {
				doc.Document.ReplacementCode, err = identification.NewGenerationCode()
				assert.NoError(t, err)
			}

			got := (&strategy.InvalidationReasonStrategy{Document: doc}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, got)
				return
			}
			if assert.NotNil(t, got) {
				test.AssertErrorCode(t, got, tc.wantCode)
			}
		})
	}
}

// TestAddMonthsClamped covers month additions that land on a shorter month.
func TestAddMonthsClamped(t *testing.T) {
	cases := []struct {
		from   string
		months int
		want   string
	}{
		{"2026-11-30", 3, "2027-02-28"},
		{"2027-11-30", 3, "2028-02-29"},
		{"2027-01-31", 1, "2027-02-28"},
		{"2028-01-31", 1, "2028-02-29"},
		{"2026-03-15", 3, "2026-06-15"},
	}
	for _, tc := range cases {
		t.Run(tc.from, func(t *testing.T) {
			assert.Equal(t, tc.want, calendar.AddMonthsClamped(day(tc.from), tc.months).Format("2006-01-02"))
		})
	}
}
