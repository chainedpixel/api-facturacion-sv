package strategy

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	commonStrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/validator/strategy"
)

type CCFRelatedDocStrategy struct {
	Document *ccf_models.CreditFiscalDocument
}

// Validate - Validates the related document types and the links between the items and the related documents
func (s *CCFRelatedDocStrategy) Validate() *dte_errors.DTEError {
	relatedDocs := s.Document.GetRelatedDocuments()

	if len(relatedDocs) > 50 {
		return dte_errors.NewDTEErrorSimple("ExceededRelatedDocsLimit", len(relatedDocs))
	}

	for _, doc := range relatedDocs {
		if err := s.validateRelatedDocType(doc.GetDocumentType()); err != nil {
			return err
		}
	}

	return commonStrategy.ValidateItemRelatedDocs(relatedDocs, s.Document.CreditItems)
}

// validateRelatedDocType - Validates that the related document type is valid for a Fiscal Credit Voucher
func (s *CCFRelatedDocStrategy) validateRelatedDocType(docType string) *dte_errors.DTEError {

	if !constants.ValidCCFDTETypesRelateDoc[docType] {
		return dte_errors.NewDTEErrorSimple("InvalidRelatedDocDTEType", docType, constants.ShowValidRelatedDocTypes(constants.ValidCCFDTETypesRelateDoc))
	}

	return nil
}
