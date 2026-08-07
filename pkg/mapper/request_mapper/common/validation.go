package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// ValidateRelatedDocs validates a slice of related document requests, ensuring each entry
// has the required fields and a valid emission date for physical documents.
func ValidateRelatedDocs(docs []structs.RelatedDocRequest) error {
	for _, doc := range docs {
		if doc.DocumentType == "" {
			return dte_errors.NewValidationError("RequiredField", "Request->RelatedDocs->DocumentType")
		}
		if doc.DocumentNumber == "" {
			return dte_errors.NewValidationError("RequiredField", "Request->RelatedDocs->DocumentNumber")
		}
		if doc.GenerationType == 0 {
			return dte_errors.NewValidationError("RequiredField", "Request->RelatedDocs->GenerationType")
		}
		if doc.GenerationType == constants.PhysicalDocument && doc.EmissionDate == "" {
			return dte_errors.NewValidationError("InvalidEmissionDateForPhysicalDocument", doc.EmissionDate)
		}
	}
	return nil
}
