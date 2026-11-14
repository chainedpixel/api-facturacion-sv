package strategy

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	commonStrategy "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invoice/invoice_models"
)

// InvoiceRelatedDocStrategy validates the related documents of an electronic invoice.
type InvoiceRelatedDocStrategy struct {
	Document *invoice_models.ElectronicInvoice
}

// Validate validates the related document types and the links between the items and the related documents.
func (s *InvoiceRelatedDocStrategy) Validate() *dte_errors.DTEError {
	relatedDocs := s.Document.GetRelatedDocuments()

	if len(relatedDocs) > 50 {
		return dte_errors.NewDTEErrorSimple("ExceededRelatedDocsLimit", len(relatedDocs))
	}

	for _, doc := range relatedDocs {
		if err := s.validateRelatedDocType(doc.GetDocumentType()); err != nil {
			return err
		}
	}

	return commonStrategy.ValidateItemRelatedDocs(relatedDocs, s.Document.InvoiceItems)
}

// validateRelatedDocType validates that the related document type is valid for an electronic invoice.
func (s *InvoiceRelatedDocStrategy) validateRelatedDocType(docType string) *dte_errors.DTEError {

	if !constants.ValidInvoiceDTETypesRelateDoc[docType] {
		return dte_errors.NewDTEErrorSimple("InvalidRelatedDocDTEType", docType, constants.ShowValidRelatedDocTypes(constants.ValidInvoiceDTETypesRelateDoc))
	}

	return nil
}
