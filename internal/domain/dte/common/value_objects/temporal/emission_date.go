package temporal

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type EmissionDate struct {
	Value time.Time `json:"value"`
}

func NewEmissionDate(value time.Time) (*EmissionDate, error) {
	date := &EmissionDate{Value: value}
	if date.IsValid() {
		return date, nil
	}
	return &EmissionDate{}, dte_errors.NewValidationError("InvalidDateTime", value.String())
}

func NewEmissionDateFromString(value string) (*EmissionDate, error) {
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, dte_errors.NewValidationError("InvalidDateTime", value)
	}
	return NewEmissionDate(date)
}

func NewValidatedEmissionDate(value time.Time) *EmissionDate {
	return &EmissionDate{Value: value}
}

// IsValid validates that the date is not zero and is before the current date
func (ed *EmissionDate) IsValid() bool {
	return !ed.Value.IsZero() && ed.Value.Before(utils.TimeNow().Add(time.Hour*24))
}

func (ed *EmissionDate) Equals(other interfaces.ValueObject[time.Time]) bool {
	return ed.GetValue().Equal(other.GetValue())
}

func (ed *EmissionDate) GetValue() time.Time {
	return ed.Value
}

func (ed *EmissionDate) ToString() string {
	return ed.Value.Format("2006-01-02")
}
