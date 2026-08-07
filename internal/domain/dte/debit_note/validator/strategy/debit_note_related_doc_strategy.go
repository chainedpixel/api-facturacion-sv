package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type DebitNoteRelatedDocStrategy struct {
	Document *debit_note_models.DebitNoteModel
}

func (s *DebitNoteRelatedDocStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return dte_errors.NewDTEErrorSimple("NullDocument", "DebitNoteRelatedDocStrategy")
	}

	if s.Document.GetRelatedDocuments() == nil || len(s.Document.GetRelatedDocuments()) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredField", "RelatedDocuments")
	}

	if len(s.Document.GetRelatedDocuments()) > 50 {
		return dte_errors.NewDTEErrorSimple("ExceededRelatedDocsLimit",
			len(s.Document.GetRelatedDocuments()))
	}

	for _, doc := range s.Document.GetRelatedDocuments() {
		if err := s.validateRelatedDocType(doc.GetDocumentType()); err != nil {
			return err
		}
	}

	for _, item := range s.Document.DebitItems {
		if item.GetRelatedDoc() == nil {
			logs.Error("Missing related document in item", map[string]interface{}{
				"itemNumber": item.GetNumber(),
			})
			return dte_errors.NewDTEErrorSimple("MissingItemRelatedDoc", item.GetNumber())
		}

		found := false
		itemRelatedDoc := *item.GetRelatedDoc()

		for _, relDoc := range s.Document.GetRelatedDocuments() {
			if relDoc.GetDocumentNumber() == itemRelatedDoc {
				found = true
				break
			}
		}

		if !found {
			logs.Error("Item related document not found in document related docs", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"relatedDoc": itemRelatedDoc,
			})
			return dte_errors.NewDTEErrorSimple("InvalidItemRelatedDoc",
				item.GetNumber(),
				itemRelatedDoc)
		}
	}

	return nil
}

func (s *DebitNoteRelatedDocStrategy) validateRelatedDocType(docType string) *dte_errors.DTEError {
	if !constants.ValidAdjustmentDTETypes[docType] {
		return dte_errors.NewDTEErrorSimple("InvalidRelatedDocTypeForCreditNote", docType)
	}

	return nil
}
