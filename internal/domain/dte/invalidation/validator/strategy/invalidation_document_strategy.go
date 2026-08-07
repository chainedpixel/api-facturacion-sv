package strategy

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type InvalidationDocumentStrategy struct {
	Document *invalidation_models.InvalidationDocument
}

func (s *InvalidationDocumentStrategy) Validate() *dte_errors.DTEError {
	doc := s.Document.Document
	if doc == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Document to invalidate")
	}

	if doc.Type.GetValue() == "" || doc.GenerationCode.GetValue() == "" || doc.ReceptionStamp == "" ||
		doc.ControlNumber.GetValue() == "" || doc.EmissionDate.GetValue().IsZero() {
		logs.Info("DEBUG", map[string]interface{}{
			"docType":        doc.Type.GetValue(),
			"generationCode": doc.GenerationCode.GetValue(),
			"receptionStamp": doc.ReceptionStamp,
			"controlNumber":  doc.ControlNumber.GetValue(),
			"emissionDate":   doc.EmissionDate.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("RequiredField", "Document")
	}

	if matched, _ := regexp.MatchString("^[A-Z0-9]{40}$", doc.ReceptionStamp); !matched {
		return dte_errors.NewDTEErrorSimple("InvalidPattern", "Reception stamp", "40 caracteres alfanumericos", doc.ReceptionStamp)
	}

	_, err := document.NewDTEType(doc.Type.GetValue())
	if err != nil {
		return dte_errors.NewDTEErrorSimple("InvalidDTETypeForInvalidation", doc.Type.GetValue())
	}

	return nil
}
