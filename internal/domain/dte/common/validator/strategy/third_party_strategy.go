package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ThirdPartyStrategy struct {
	Document interfaces.DTEDocument
}

// Validate validates that the DTE items are consistent with a third-party sale or that there is no third-party sale
func (s *ThirdPartyStrategy) Validate() *dte_errors.DTEError {
	if s.Document.GetThirdPartySale() == nil {
		return nil
	}

	// Anexo IV, seccion 6: las ventas por cuenta de terceros van en documentos separados de las propias;
	// al informarse ventaTercero, todo el documento corresponde al tercero
	if err := s.validateSingleThirdParty(); err != nil {
		return err
	}

	return nil
}

// validateSingleThirdParty validates that there is only one third party per DTE
func (s *ThirdPartyStrategy) validateSingleThirdParty() *dte_errors.DTEError {
	if s.Document.GetThirdPartySale() == nil {
		return nil
	}

	if s.Document.GetThirdPartySale().GetName() == "" {
		return dte_errors.NewDTEErrorSimple("RequiredField", "ThirdPartyName")
	}

	return nil
}
