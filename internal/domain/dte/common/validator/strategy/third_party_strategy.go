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

	for _, item := range s.Document.GetItems() {
		if err := s.validateThirdPartyItem(item); err != nil {
			return err
		}
	}

	if err := s.validateSingleThirdParty(); err != nil {
		return err
	}

	return nil
}

// validateThirdPartyItem validates each DTE item to ensure all are related to a third-party sale
func (s *ThirdPartyStrategy) validateThirdPartyItem(item interfaces.Item) *dte_errors.DTEError {
	if s.Document.GetThirdPartySale() != nil {
		if item.GetRelatedDoc() == nil {
			return dte_errors.NewDTEErrorSimple("MixedSalesNotAllowed")
		}
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
