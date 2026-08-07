package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ModelTypeStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Validates the model type rules of a DTE document
func (s *ModelTypeStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetIdentification() == nil {
		return nil
	}

	if s.Document.GetIdentification().GetOperationType() == constants.TransmisionNormal &&
		s.Document.GetIdentification().GetModelType() != constants.ModeloFacturacionPrevio {
		return dte_errors.NewDTEErrorSimple("InvalidModelType",
			s.Document.GetIdentification().GetModelType())
	}

	if s.Document.GetIdentification().GetOperationType() == constants.TransmisionContingencia &&
		s.Document.GetIdentification().GetModelType() != constants.ModeloFacturacionDiferido {
		return dte_errors.NewDTEErrorSimple("InvalidModelType",
			s.Document.GetIdentification().GetModelType())
	}

	return nil
}
