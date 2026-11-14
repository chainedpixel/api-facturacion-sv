package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	commonStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
)

// RemissionNoteRelatedDocStrategy validates the links between the remission note items and its related documents.
type RemissionNoteRelatedDocStrategy struct {
	Document *remission_note_models.RemissionNoteModel
}

// Validate checks that every related document is referenced by at least one item and that no item references a
// document outside the related documents section.
func (s *RemissionNoteRelatedDocStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || s.Document.DTEDocument == nil {
		return nil
	}

	return commonStrategy.ValidateItemRelatedDocs(s.Document.GetRelatedDocuments(), s.Document.RemissionItems)
}
