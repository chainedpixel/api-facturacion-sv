package policy

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/calendar"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
)

const businessDaysAfterPeriod = 10

var pharmaceuticalActivities = map[string]bool{
	"21001": true,
	"21008": true,
	"46482": true,
	"46484": true,
	"46491": true,
	"47721": true,
}

// RequiresReplacement reports whether an invalidation of type 1 or 3 must carry a replacement generation
// code; credit notes and settlement vouchers must leave it null.
func RequiresReplacement(dteType string) bool {
	return dteType != constants.NotaCreditoElectronica && dteType != constants.ComprobanteLiquidacionElectronico
}

// IsInvoiceType reports whether the DTE is an FE, FEXE or FSEE, whose invalidation deadline is counted in
// months from the reception stamp and whose event date may be later than the generation date.
func IsInvoiceType(dteType string) bool {
	return dteType == constants.FacturaElectronica ||
		dteType == constants.FacturaExportacionElectronica ||
		dteType == constants.FacturaSujetoExcluidoElectronica
}

// Deadlines computes the invalidation deadline of each DTE type.
type Deadlines struct {
	calendar *calendar.BusinessCalendar
}

// NewDeadlines creates a Deadlines calculator; a nil calendar uses national holidays only.
func NewDeadlines(businessCalendar *calendar.BusinessCalendar) *Deadlines {
	if businessCalendar == nil {
		businessCalendar = calendar.NewBusinessCalendar(nil)
	}
	return &Deadlines{calendar: businessCalendar}
}

// TransmissionDeadline returns the last instant to transmit the invalidation event: the end of the day
// three months after the stamp for FE, FEXE and FSEE (two years for pharmaceutical FE and FEXE issuers),
// otherwise the end of the tenth business day of the month following the stamp.
func (d *Deadlines) TransmissionDeadline(dteType string, stampDate time.Time, issuerActivity string) time.Time {
	if IsInvoiceType(dteType) {
		return endOfDay(AddValidityPeriod(dteType, stampDate, issuerActivity))
	}

	nextMonth := time.Date(stampDate.Year(), stampDate.Month(), 1, 0, 0, 0, 0, stampDate.Location()).AddDate(0, 1, 0)
	tenthBusinessDay := d.calendar.NthBusinessDayOfMonth(nextMonth.Year(), nextMonth.Month(), businessDaysAfterPeriod, stampDate.Location())
	return endOfDay(tenthBusinessDay)
}

// AddValidityPeriod adds the FE, FEXE or FSEE invalidation period (three months or two years) to a date,
// keeping the last day of the target month when the day does not exist in it.
func AddValidityPeriod(dteType string, from time.Time, issuerActivity string) time.Time {
	if dteType != constants.FacturaSujetoExcluidoElectronica && pharmaceuticalActivities[issuerActivity] {
		return calendar.AddMonthsClamped(from, 24)
	}
	return calendar.AddMonthsClamped(from, 3)
}

// EventDateTime returns the invalidation event date and time: the current instant for FE, FEXE and FSEE;
// for any other DTE the generation date and time of the document, or the current instant when the
// document was generated today.
func EventDateTime(dteType string, generation, now time.Time) time.Time {
	if IsInvoiceType(dteType) || generation.IsZero() || SameDay(generation, now) {
		return now
	}
	return generation
}

// SameDay reports whether both instants fall on the same calendar day in the location of the first one.
func SameDay(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.In(a.Location()).Format("2006-01-02")
}

func endOfDay(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, day.Location())
}
