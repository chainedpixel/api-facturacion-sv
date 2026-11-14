package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/policy"
)

type InvalidationReasonStrategy struct {
	Document *invalidation_models.InvalidationDocument
}

func (s *InvalidationReasonStrategy) Validate() *dte_errors.DTEError {
	reason := s.Document.Reason
	if reason == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Reason")
	}

	if reason.ResponsibleName == "" ||
		reason.ResponsibleDocType.GetValue() == "" ||
		reason.ResponsibleDocNum.GetValue() == "" ||
		reason.RequesterName == "" ||
		reason.RequesterDocType.GetValue() == "" ||
		reason.RequesterDocNum.GetValue() == "" {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Reason required fields")
	}

	if reason.Type.GetValue() < 1 || reason.Type.GetValue() > 3 {
		return dte_errors.NewDTEErrorSimple("InvalidEnum", "Annulment type")
	}

	if !contains(reason.ResponsibleDocType.GetValue()) ||
		!contains(reason.RequesterDocType.GetValue()) {
		return dte_errors.NewDTEErrorSimple("InvalidEnum", "Document type")
	}

	if reason.Type.GetValue() == 3 && reason.Reason == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Reason for type 3")
	}

	if reason.Type.GetValue() == 2 {
		if s.Document.Document.ReplacementCode != nil {
			return dte_errors.NewDTEErrorSimple("InvalidInvalidationType2")
		}
	} else {
		requiresReplacement := policy.RequiresReplacement(s.Document.Document.Type.GetValue())
		replacement := s.Document.Document.ReplacementCode
		if requiresReplacement && replacement == nil {
			return dte_errors.NewDTEErrorSimple("InvalidReplacementCode1And3")
		}
		if !requiresReplacement && replacement != nil {
			return dte_errors.NewDTEErrorSimple("ReplacementCodeNotAllowedForType", s.Document.Document.Type.GetValue())
		}
	}

	return nil
}

func contains(item string) bool {
	for _, s := range constants.ValidReceiverDTETypes {
		if s == item {
			return true
		}
	}
	return false
}
