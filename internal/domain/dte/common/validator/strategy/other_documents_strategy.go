package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type OtherDocumentsStrategy struct {
	Document interfaces.DTEDocument
}

// Validate validates the additional documents of the DTE
func (s *OtherDocumentsStrategy) Validate() *dte_errors.DTEError {
	docs := s.Document.GetOtherDocuments()
	if docs == nil || len(docs) == 0 {
		return nil
	}

	if len(docs) > 10 {
		return dte_errors.NewDTEErrorSimple("InvalidOtherDocsCount", len(docs))
	}

	for _, doc := range docs {
		if err := s.validateDocument(doc); err != nil {
			return err
		}
	}

	return nil
}

// validateDocument validates a document
func (s *OtherDocumentsStrategy) validateDocument(doc interfaces.OtherDocuments) *dte_errors.DTEError {
	code := doc.GetAssociatedDocument()
	if code < 1 || code > 4 {
		return dte_errors.NewDTEErrorSimple("InvalidAssociatedDocumentCode", code)
	}

	if code == 3 {
		return s.validateMedicalDocument(doc)
	}

	return s.validateRegularDocument(doc)
}

// validateMedicalDocument validates a medical document
func (s *OtherDocumentsStrategy) validateMedicalDocument(doc interfaces.OtherDocuments) *dte_errors.DTEError {
	if doc.GetDescription() != "" || doc.GetDetail() != "" {
		return dte_errors.NewDTEErrorSimple("InvalidMedicalDocFields")
	}

	if doc.GetDoctor() == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "OtherDocuments->Doctor")
	}

	return s.validateDoctor(doc.GetDoctor())
}

// validateRegularDocument validates a regular (non-medical) document
func (s *OtherDocumentsStrategy) validateRegularDocument(doc interfaces.OtherDocuments) *dte_errors.DTEError {
	if doc.GetDoctor().GetName() != "" || doc.GetDoctor().GetServiceType() != 0 || doc.GetDoctor().GetNIT() != "" || doc.GetDoctor().GetIdentification() != "" {
		return dte_errors.NewDTEErrorSimple("InvalidField", "Doctor must be null")
	}

	if doc.GetAssociatedDocument() != 3 {
		if doc.GetDescription() == "" {
			return dte_errors.NewDTEErrorSimple("RequiredField", "OtherDocuments->Description")
		}

		if doc.GetDetail() == "" {
			return dte_errors.NewDTEErrorSimple("RequiredField", "OtherDocuments->Detail")
		}
	}

	if len(doc.GetDescription()) > 100 {
		return dte_errors.NewDTEErrorSimple("InvalidLength", "Description", "1-100", doc.GetDescription())
	}

	if len(doc.GetDetail()) > 300 {
		return dte_errors.NewDTEErrorSimple("InvalidLength", "Detail", "1-300", doc.GetDetail())
	}

	return nil
}

// validateDoctor validates the doctor data in a medical document
func (s *OtherDocumentsStrategy) validateDoctor(doctor interfaces.DoctorInfo) *dte_errors.DTEError {
	if len(doctor.GetName()) == 0 || len(doctor.GetName()) > 100 {
		return dte_errors.NewDTEErrorSimple("InvalidLength", "DoctorName", "1-100", doctor.GetName())
	}

	serviceType := doctor.GetServiceType()
	if serviceType < 1 || serviceType > 6 {
		return dte_errors.NewDTEErrorSimple("InvalidServiceType", serviceType)
	}

	hasNIT := doctor.GetNIT() != ""
	hasID := doctor.GetIdentification() != ""

	if !hasNIT && !hasID {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Doctor NIT or Identification")
	}

	if hasNIT && hasID {
		return dte_errors.NewDTEErrorSimple("MutuallyExclusiveFields", "NIT", "Identification")
	}

	return nil
}
