package strategy

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type RelatedDocsStrategy struct {
	Document interfaces.DTEDocument
}

// Validate validates the related documents strategy of the DTE
func (s *RelatedDocsStrategy) Validate() *dte_errors.DTEError {
	if s.Document.GetRelatedDocuments() == nil || len(s.Document.GetRelatedDocuments()) == 0 {
		return nil
	}

	if len(s.Document.GetRelatedDocuments()) > 50 {
		return dte_errors.NewDTEErrorSimple("ExceededRelatedDocsLimit",
			len(s.Document.GetRelatedDocuments()))
	}

	firstDocType := s.Document.GetRelatedDocuments()[0].GetDocumentType()
	for i, doc := range s.Document.GetRelatedDocuments() {
		if doc.GetDocumentType() != firstDocType {
			logs.Error("Mixed document types not allowed", map[string]interface{}{
				"expectedType": firstDocType,
				"foundType":    doc.GetDocumentType(),
				"index":        i,
			})
			return dte_errors.NewDTEErrorSimple("MixedDocumentTypesNotAllowed")
		}
	}

	for _, doc := range s.Document.GetRelatedDocuments() {
		if err := s.validateRelatedDoc(doc); err != nil {
			return err
		}
	}

	return nil
}

// validateRelatedDoc validates a related document
func (s *RelatedDocsStrategy) validateRelatedDoc(doc interfaces.RelatedDocument) *dte_errors.DTEError {

	if doc.GetEmissionDate().After(utils.TimeNow()) {
		return dte_errors.NewDTEErrorSimple("InvalidRelatedDocDate",
			doc.GetEmissionDate().Format("2006-01-02"))
	}

	err := validateElectronicDocNumber(doc.GetDocumentNumber(), doc.GetGenerationType())
	if err != nil {
		return err
	}

	return nil
}

// validateElectronicDocNumber validates the electronic document number
func validateElectronicDocNumber(number string, generationType int) *dte_errors.DTEError {

	if generationType == constants.TransmisionContingencia {
		if !isValidUUID(number) {
			return dte_errors.NewDTEErrorSimple("InvalidRelatedDocNumberContingency", number)
		}
	} else {
		if len(number) < 0 || len(number) > 20 {
			return dte_errors.NewDTEErrorSimple("InvalidRelatedDocNumberNormal", number)
		}
	}

	return nil
}

// isValidUUID validates that the related document number is a valid UUID
func isValidUUID(uuid string) bool {
	var uuidRegex = regexp.MustCompile(`^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$`)
	return uuidRegex.MatchString(uuid)
}
