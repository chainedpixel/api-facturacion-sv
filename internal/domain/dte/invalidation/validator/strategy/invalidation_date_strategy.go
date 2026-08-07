package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
)

type InvalidationDateStrategy struct {
	Document *invalidation_models.InvalidationDocument
}

func (s *InvalidationDateStrategy) Validate() *dte_errors.DTEError {
	if s.Document.Identification == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Identification")
	}

	docType := s.Document.Document.Type.GetValue()
	emissionDate := s.Document.Document.EmissionDate.GetValue()
	annulmentDate := s.Document.Identification.EmissionDate.GetValue()

	switch docType {
	case "01", "11":
		if annulmentDate.Sub(emissionDate).Hours() > 24*90 {
			return dte_errors.NewDTEErrorSimple("InvalidDateForFEFX")
		}
	default:
		if annulmentDate.Sub(emissionDate).Hours() > 24 {
			return dte_errors.NewDTEErrorSimple("InvalidDateForAllDTE")
		}
	}
	return nil
}
