package strategy

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
)

// ValidateItemRelatedDocs checks the links between the items and the related documents section: when the section
// has documents every item must reference one of them and every document must be referenced by at least one item;
// without documents no item may carry a reference.
func ValidateItemRelatedDocs[T interfaces.ItemGetter](relatedDocs []interfaces.RelatedDocument, items []T) *dte_errors.DTEError {
	referenced := make(map[string]bool, len(relatedDocs))
	for _, doc := range relatedDocs {
		referenced[doc.GetDocumentNumber()] = false
	}

	for _, item := range items {
		relatedDoc := item.GetRelatedDoc()
		if relatedDoc == nil {
			if len(relatedDocs) > 0 {
				return dte_errors.NewDTEErrorSimple("MissingItemRelatedDoc", item.GetNumber())
			}
			continue
		}

		if _, exists := referenced[*relatedDoc]; !exists {
			return dte_errors.NewDTEErrorSimple("InvalidItemRelatedDoc", item.GetNumber(), *relatedDoc)
		}
		referenced[*relatedDoc] = true
	}

	for _, doc := range relatedDocs {
		if !referenced[doc.GetDocumentNumber()] {
			return dte_errors.NewDTEErrorSimple("UnreferencedRelatedDoc", doc.GetDocumentNumber())
		}
	}

	return nil
}
