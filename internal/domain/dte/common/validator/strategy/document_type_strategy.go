package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type DocumentTypeStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Validates the document type rules of a DTE document
func (s *DocumentTypeStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetIdentification() == nil {
		return nil
	}

	docType := s.Document.GetIdentification().GetDTEType()

	switch docType {
	case constants.CCFElectronico:
		if s.Document.GetReceiver().GetNRC() == nil {
			return dte_errors.NewDTEErrorSimple("MissingNRC",
				s.Document.GetReceiver().GetName(),
				constants.CCFElectronico)
		}
	}

	return nil
}
