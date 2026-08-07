package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type CCFReceiverStrategy struct {
	Document *ccf_models.CreditFiscalDocument
}

// Validate - Validates the specific fields of a CCF receiver
func (s *CCFReceiverStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.GetReceiver() == nil {
		return nil
	}

	if s.Document.GetReceiver().GetNRC() == nil {
		return dte_errors.NewDTEErrorSimple("MissingNRC",
			utils.PointerToString(s.Document.GetReceiver().GetName()),
			constants.CCFElectronico)
	}

	if s.Document.GetReceiver().GetActivityCode() == nil ||
		s.Document.GetReceiver().GetActivityDescription() == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField",
			"ActivityCode and Description")
	}

	return nil
}
