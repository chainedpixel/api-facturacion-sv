package strategy

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type TemporalValidationStrategy struct {
	Document interfaces.DTEDocument
}

// Validate validates the emission date and time of the document
func (s *TemporalValidationStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetIdentification() == nil {
		return nil
	}

	emissionDate := s.Document.GetIdentification().GetEmissionDate()
	emissionTime := s.Document.GetIdentification().GetEmissionTime()
	now := utils.TimeNow()

	if emissionDate.After(now) {
		return dte_errors.NewDTEErrorSimple("InvalidDateTime",
			emissionDate.Format("2006-01-02"))
	}

	if emissionDate.Equal(now.Truncate(24*time.Hour)) &&
		emissionTime.After(now) {
		return dte_errors.NewDTEErrorSimple("InvalidEmissionTime",
			emissionTime.Format("15:04:05"))
	}

	return nil
}
