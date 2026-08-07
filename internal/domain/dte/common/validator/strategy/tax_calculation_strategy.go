package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type TaxCalculationStrategy struct {
	Document interfaces.DTEDocument
}

// Validate validates that the tax codes are valid in the document items
func (s *TaxCalculationStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.GetItems()) == 0 {
		return nil
	}

	for _, item := range s.Document.GetItems() {
		if err := s.validateItemTaxCodes(item); err != nil {
			return err
		}
	}

	return nil
}

// validateItemTaxCodes validates that the tax codes are valid
func (s *TaxCalculationStrategy) validateItemTaxCodes(item interfaces.Item) *dte_errors.DTEError {
	for _, tax := range item.GetTaxes() {
		if exist := constants.MapAllowedTaxTypes[tax]; !exist {
			return dte_errors.NewDTEErrorSimple("UnsupportedTaxCode", tax)
		}
	}
	return nil
}
