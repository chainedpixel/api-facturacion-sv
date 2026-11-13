package calendar

import (
	"fmt"
	"strings"
	"time"
)

const (
	dateLayout               = "2006-01-02"
	holyWeekDaysBeforeEaster = 6
	holyWeekDaysAfterEaster  = 1
)

var fixedHolidays = map[string]bool{
	"01-01": true,
	"05-01": true,
	"05-10": true,
	"06-17": true,
	"08-06": true,
	"09-15": true,
	"11-02": true,
	"12-25": true,
}

// BusinessCalendar resolves business days for El Salvador: Monday to Friday, excluding fixed national
// holidays, Holy Week from Holy Monday through Easter Monday, and any configured extra holiday.
type BusinessCalendar struct {
	extraHolidays map[string]bool
}

// NewBusinessCalendar creates a calendar with the given extra holidays.
func NewBusinessCalendar(extraHolidays []time.Time) *BusinessCalendar {
	extra := make(map[string]bool, len(extraHolidays))
	for _, day := range extraHolidays {
		extra[day.Format(dateLayout)] = true
	}
	return &BusinessCalendar{extraHolidays: extra}
}

// ParseHolidays parses a comma separated list of YYYY-MM-DD dates. An empty string yields no holidays.
func ParseHolidays(raw string) ([]time.Time, error) {
	var holidays []time.Time
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		day, err := time.Parse(dateLayout, value)
		if err != nil {
			return nil, fmt.Errorf("invalid holiday %q, expected format YYYY-MM-DD", value)
		}
		holidays = append(holidays, day)
	}
	return holidays, nil
}

// IsHoliday reports whether the day is a national, Holy Week or configured extra holiday.
func (c *BusinessCalendar) IsHoliday(day time.Time) bool {
	if fixedHolidays[day.Format("01-02")] || c.extraHolidays[day.Format(dateLayout)] {
		return true
	}
	return isHolyWeekHoliday(day)
}

// IsBusinessDay reports whether the day is a weekday that is not a holiday.
func (c *BusinessCalendar) IsBusinessDay(day time.Time) bool {
	weekday := day.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday && !c.IsHoliday(day)
}

// NthBusinessDayOfMonth returns the nth business day of the given month in the given location.
func (c *BusinessCalendar) NthBusinessDayOfMonth(year int, month time.Month, n int, loc *time.Location) time.Time {
	day := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	count := 0
	for {
		if c.IsBusinessDay(day) {
			count++
			if count == n {
				return day
			}
		}
		day = day.AddDate(0, 0, 1)
	}
}

func isHolyWeekHoliday(day time.Time) bool {
	easter := easterSunday(day.Year(), day.Location())
	date := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	holyMonday := easter.AddDate(0, 0, -holyWeekDaysBeforeEaster)
	easterMonday := easter.AddDate(0, 0, holyWeekDaysAfterEaster)
	return !date.Before(holyMonday) && !date.After(easterMonday)
}

func easterSunday(year int, loc *time.Location) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
}
