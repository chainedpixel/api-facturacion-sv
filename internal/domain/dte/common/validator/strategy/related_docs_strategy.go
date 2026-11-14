package strategy

import (
	"regexp"
	"time"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/logs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
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

	dteType, generation := s.documentTypeAndGeneration()
	for _, doc := range s.Document.GetRelatedDocuments() {
		if err := s.validateRelatedDoc(doc); err != nil {
			return err
		}
		if err := ValidateRelatedDocType(dteType, doc.GetDocumentType()); err != nil {
			return err
		}
		if err := ValidateRelatedDocPeriod(dteType, doc.GetDocumentType(), doc.GetEmissionDate(), generation); err != nil {
			return err
		}
	}

	return nil
}

func (s *RelatedDocsStrategy) documentTypeAndGeneration() (string, time.Time) {
	identification := s.Document.GetIdentification()
	if identification == nil {
		return "", utils.TimeNow()
	}
	generation := identification.GetEmissionDate()
	if generation.IsZero() {
		generation = utils.TimeNow()
	}
	return identification.GetDTEType(), generation
}

// validateRelatedDoc validates a related document
func (s *RelatedDocsStrategy) validateRelatedDoc(doc interfaces.RelatedDocument) *dte_errors.DTEError {

	// La normativa compara fechas (dia), no horas: se evita rechazar por diferencias de reloj entre la BD y la API
	now := utils.TimeNow()
	docDay := doc.GetEmissionDate().In(now.Location()).Format("2006-01-02")
	if docDay > now.Format("2006-01-02") {
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

	if generationType == constants.ElectronicDocument {
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
