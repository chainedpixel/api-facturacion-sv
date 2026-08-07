package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ContingencyStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Validates the contingency rules of a DTE document
func (s *ContingencyStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetIdentification() == nil {
		return nil
	}

	if s.Document.GetIdentification().GetOperationType() == constants.TransmisionContingencia {
		if s.Document.GetIdentification().GetContingencyType() == nil {
			return dte_errors.NewDTEErrorSimple("MissingContingencyType")
		}

		if *s.Document.GetIdentification().GetContingencyType() == constants.OtroMotivo &&
			len(*s.Document.GetIdentification().GetContingencyReason()) == 0 {
			return dte_errors.NewDTEErrorSimple("MissingContingencyReason")
		}

		if !validateContingencyType(*s.Document.GetIdentification().GetContingencyType()) {
			return dte_errors.NewDTEErrorSimple("InvalidContingencyType",
				s.Document.GetIdentification().GetContingencyType())
		}
	}

	return nil
}

// validateContingencyType Validates that the contingency type is one of the allowed values
func validateContingencyType(contingencyType int) bool {
	for _, ct := range constants.AllowedContingencyTypes {
		if ct == contingencyType {
			return true
		}
	}
	return false
}
