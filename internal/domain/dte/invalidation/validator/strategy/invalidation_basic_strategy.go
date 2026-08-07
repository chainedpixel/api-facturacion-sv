package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
)

type InvalidationBasicStrategy struct {
	Document *invalidation_models.InvalidationDocument
}

func (s *InvalidationBasicStrategy) Validate() *dte_errors.DTEError {
	if err := s.validateIdentification(); err != nil {
		return err
	}

	if err := s.validateIssuer(); err != nil {
		return err
	}

	return nil
}

func (s *InvalidationBasicStrategy) validateIdentification() *dte_errors.DTEError {
	id := s.Document.Identification
	if id == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Identification")
	}

	if id.Version.GetValue() == 0 || id.Ambient.GetValue() == "" || id.GenerationCode.GetValue() == "" ||
		id.GetEmissionDate().IsZero() || id.GetEmissionTime().IsZero() {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Identification fields")
	}

	return nil
}

func (s *InvalidationBasicStrategy) validateIssuer() *dte_errors.DTEError {
	issuer := s.Document.Issuer
	if issuer == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Issuer")
	}

	if issuer.NIT.GetValue() == "" || issuer.Name == "" || issuer.EstablishmentType.GetValue() == "" {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Issuer required fields")
	}

	if issuer.Email.GetValue() == "" {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Issuer email")
	}

	return nil
}
