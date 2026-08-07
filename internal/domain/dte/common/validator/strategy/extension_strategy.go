package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ExtensionStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Validates the extension rules of a DTE document
func (s *ExtensionStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetSummary() == nil {
		return nil
	}

	if s.Document.GetSummary().GetTotalOperation() >= 1095.00 {
		if s.Document.GetExtension() == nil {
			return dte_errors.NewDTEErrorSimple("RequiredExtension", s.Document.GetSummary().GetTotalOperation())
		}
	}
	return nil
}
