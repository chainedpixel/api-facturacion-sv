package services

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/calendar"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/policy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/validator/strategy"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
)

func svDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, utils.TimeNow().Location())
	if err != nil {
		t.Fatalf("invalid date %s: %v", value, err)
	}
	return parsed
}

func TestBusinessCalendar(t *testing.T) {
	test.TestMain(t)
	cal := calendar.NewBusinessCalendar(nil)
	loc := utils.TimeNow().Location()

	t.Run("Fixed national and Holy Week holidays", func(t *testing.T) {
		assert.True(t, cal.IsHoliday(svDate(t, "2027-01-01 00:00:00")))
		assert.True(t, cal.IsHoliday(svDate(t, "2026-09-15 00:00:00")))
		assert.True(t, cal.IsHoliday(svDate(t, "2027-03-22 00:00:00")), "Holy Monday")
		assert.True(t, cal.IsHoliday(svDate(t, "2027-03-25 00:00:00")), "Holy Thursday")
		assert.True(t, cal.IsHoliday(svDate(t, "2027-03-26 00:00:00")), "Holy Friday")
		assert.True(t, cal.IsHoliday(svDate(t, "2027-03-29 00:00:00")), "Easter Monday")
		assert.False(t, cal.IsHoliday(svDate(t, "2027-03-19 00:00:00")))
		assert.False(t, cal.IsHoliday(svDate(t, "2027-03-30 00:00:00")))
	})

	t.Run("Weekend is not a business day", func(t *testing.T) {
		assert.False(t, cal.IsBusinessDay(svDate(t, "2026-11-14 00:00:00")))
		assert.True(t, cal.IsBusinessDay(svDate(t, "2026-11-13 00:00:00")))
	})

	t.Run("Tenth business day", func(t *testing.T) {
		assert.Equal(t, "2026-12-14", cal.NthBusinessDayOfMonth(2026, time.December, 10, loc).Format("2006-01-02"))
		assert.Equal(t, "2027-01-15", cal.NthBusinessDayOfMonth(2027, time.January, 10, loc).Format("2006-01-02"))
	})

	t.Run("Configured extra holiday", func(t *testing.T) {
		extra, err := calendar.ParseHolidays("2027-01-04, 2027-01-05")
		assert.NoError(t, err)
		withExtra := calendar.NewBusinessCalendar(extra)
		assert.Equal(t, "2027-01-19", withExtra.NthBusinessDayOfMonth(2027, time.January, 10, loc).Format("2006-01-02"))

		_, err = calendar.ParseHolidays("04/01/2027")
		assert.Error(t, err)
	})
}

func TestInvalidationDeadlines(t *testing.T) {
	test.TestMain(t)
	deadlines := policy.NewDeadlines(nil)
	stamp := svDate(t, "2026-11-05 10:00:00")

	tests := []struct {
		name     string
		dteType  string
		activity string
		expected string
	}{
		{"CCF: tenth business day of next month", constants.CCFElectronico, "", "2026-12-14 23:59:59"},
		{"Credit note: tenth business day of next month", constants.NotaCreditoElectronica, "", "2026-12-14 23:59:59"},
		{"Retention voucher: tenth business day of next month", constants.ComprobanteRetencionElectronico, "", "2026-12-14 23:59:59"},
		{"FE: three months from stamp", constants.FacturaElectronica, "47712", "2027-02-05 23:59:59"},
		{"FSE: three months from stamp", constants.FacturaSujetoExcluidoElectronica, "", "2027-02-05 23:59:59"},
		{"Pharmaceutical FE: two years", constants.FacturaElectronica, "47721", "2028-11-05 23:59:59"},
		{"FSE with pharmaceutical activity keeps three months", constants.FacturaSujetoExcluidoElectronica, "47721", "2027-02-05 23:59:59"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deadline := deadlines.TransmissionDeadline(tt.dteType, stamp, tt.activity)
			assert.Equal(t, tt.expected, deadline.Format("2006-01-02 15:04:05"))
		})
	}
}

// TestInvalidationDeadlinesReferenceCases checks the deadlines against published reference examples.
func TestInvalidationDeadlinesReferenceCases(t *testing.T) {
	test.TestMain(t)
	deadlines := policy.NewDeadlines(nil)

	tests := []struct {
		name     string
		dteType  string
		stamp    string
		expected string
	}{
		{"CCF stamped in October 2025", constants.CCFElectronico, "2025-10-31 10:00:00", "2025-11-14 23:59:59"},
		{"CCF stamped in March 2026 (Holy Week in April)", constants.CCFElectronico, "2026-03-19 07:30:00", "2026-04-20 23:59:59"},
		{"CCF stamped in April 2026", constants.CCFElectronico, "2026-04-25 00:30:00", "2026-05-15 23:59:59"},
		{"FE stamped on November 11, 2025", constants.FacturaElectronica, "2025-11-11 07:30:00", "2026-02-11 23:59:59"},
		{"FE stamped on October 9, 2025", constants.FacturaElectronica, "2025-10-09 08:00:00", "2026-01-09 23:59:59"},
		{"FE stamped on November 25, 2025", constants.FacturaElectronica, "2025-11-25 08:00:00", "2026-02-25 23:59:59"},
		{"FE stamped on November 30, 2025 ends on February 28", constants.FacturaElectronica, "2025-11-30 08:00:00", "2026-02-28 23:59:59"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deadline := deadlines.TransmissionDeadline(tt.dteType, svDate(t, tt.stamp), "")
			assert.Equal(t, tt.expected, deadline.Format("2006-01-02 15:04:05"))
		})
	}
}

func TestInvalidationEventDateTime(t *testing.T) {
	test.TestMain(t)
	now := svDate(t, "2026-12-08 09:15:00")

	t.Run("CCF from a previous day: generation date and time", func(t *testing.T) {
		generation := svDate(t, "2026-11-05 16:40:00")
		assert.Equal(t, generation, policy.EventDateTime(constants.CCFElectronico, generation, now))
	})

	t.Run("CCF generated today: current date and time", func(t *testing.T) {
		generation := svDate(t, "2026-12-08 08:00:00")
		assert.Equal(t, now, policy.EventDateTime(constants.CCFElectronico, generation, now))
	})

	t.Run("FE: current date and time", func(t *testing.T) {
		generation := svDate(t, "2026-11-05 16:40:00")
		assert.Equal(t, now, policy.EventDateTime(constants.FacturaElectronica, generation, now))
	})
}

func TestInvalidationDateStrategy(t *testing.T) {
	test.TestMain(t)
	deadlines := policy.NewDeadlines(nil)

	build := func(t *testing.T, dteType, generation, stamp, eventDate string) *strategy.InvalidationDateStrategy {
		doc, err := fixtures.BuildInvalidationWithAnnulment()
		if err != nil {
			t.Fatalf("fixture: %v", err)
		}
		doc.Document.Type = *document.NewValidatedDTEType(dteType)
		doc.Document.GenerationDateTime = svDate(t, generation)
		doc.Document.StampDate = svDate(t, stamp)
		doc.Identification.EmissionDate = *temporal.NewValidatedEmissionDate(svDate(t, eventDate))
		return &strategy.InvalidationDateStrategy{Document: doc, Deadlines: deadlines}
	}

	t.Run("CCF on day 8 of next month with event date equal to generation", func(t *testing.T) {
		s := build(t, constants.CCFElectronico, "2026-11-05 10:00:00", "2026-11-05 10:00:05", "2026-11-05 10:00:00")
		s.Now = svDate(t, "2026-12-08 09:00:00")
		assert.Nil(t, s.Validate())
	})

	t.Run("CCF after the tenth business day of next month", func(t *testing.T) {
		s := build(t, constants.CCFElectronico, "2026-11-05 10:00:00", "2026-11-05 10:00:05", "2026-11-05 10:00:00")
		s.Now = svDate(t, "2026-12-15 08:00:00")
		err := s.Validate()
		if assert.NotNil(t, err) {
			test.AssertErrorCode(t, err, "InvalidationDeadlineExpired")
		}
	})

	t.Run("CCF with event date different from generation", func(t *testing.T) {
		s := build(t, constants.CCFElectronico, "2026-11-05 10:00:00", "2026-11-05 10:00:05", "2026-11-06 10:00:00")
		s.Now = svDate(t, "2026-11-06 10:00:00")
		err := s.Validate()
		if assert.NotNil(t, err) {
			test.AssertErrorCode(t, err, "InvalidationEventDateMismatch")
		}
	})

	t.Run("FE two months after stamp", func(t *testing.T) {
		s := build(t, constants.FacturaElectronica, "2026-09-10 10:00:00", "2026-09-10 10:00:05", "2026-11-10 09:00:00")
		s.Now = svDate(t, "2026-11-10 09:00:00")
		assert.Nil(t, s.Validate())
	})

	t.Run("FE more than three months after stamp", func(t *testing.T) {
		s := build(t, constants.FacturaElectronica, "2026-08-10 10:00:00", "2026-08-10 10:00:05", "2026-11-11 09:00:00")
		s.Now = svDate(t, "2026-11-11 09:00:00")
		err := s.Validate()
		if assert.NotNil(t, err) {
			test.AssertErrorCode(t, err, "InvalidationDeadlineExpired")
		}
	})

	t.Run("FSE thirty days after stamp", func(t *testing.T) {
		s := build(t, constants.FacturaSujetoExcluidoElectronica, "2026-10-12 10:00:00", "2026-10-12 10:00:05", "2026-11-11 09:00:00")
		s.Now = svDate(t, "2026-11-11 09:00:00")
		assert.Nil(t, s.Validate())
	})
}

func TestInvalidationServiceValidateDependencies(t *testing.T) {
	test.TestMain(t)

	ccf := &dte.DTEDocument{Details: &dte.DTEDetails{ID: "CCF-CODE", DTEType: constants.CCFElectronico}}
	fe := &dte.DTEDocument{Details: &dte.DTEDetails{ID: "FE-CODE", DTEType: constants.FacturaElectronica}}

	t.Run("CCF with active notes is rejected listing the notes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDTE := mocks.NewMockDTEManager(ctrl)
		mockDTE.EXPECT().GetActiveAdjustmentNotes(gomock.Any(), uint(1), "CCF-CODE").
			Return([]string{"NC-1", "ND-2"}, nil)

		err := invalidation.NewInvalidationService(mockDTE, nil).ValidateDependencies(context.Background(), 1, ccf)
		test.AssertErrorCode(t, err, "PendingAdjustmentNotes")
		assert.Contains(t, err.Error(), "NC-1, ND-2")
	})

	t.Run("CCF without active notes is allowed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDTE := mocks.NewMockDTEManager(ctrl)
		mockDTE.EXPECT().GetActiveAdjustmentNotes(gomock.Any(), uint(1), "CCF-CODE").Return(nil, nil)

		assert.NoError(t, invalidation.NewInvalidationService(mockDTE, nil).ValidateDependencies(context.Background(), 1, ccf))
	})

	t.Run("FE does not query notes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDTE := mocks.NewMockDTEManager(ctrl)

		assert.NoError(t, invalidation.NewInvalidationService(mockDTE, nil).ValidateDependencies(context.Background(), 1, fe))
	})
}
