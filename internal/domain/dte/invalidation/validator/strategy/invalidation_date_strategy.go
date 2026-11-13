package strategy

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/policy"
)

const deadlineLayout = "2006-01-02 15:04:05"

// InvalidationDateStrategy validates the transmission deadline against the stamp date of the invalidated
// document and the event date against its generation date.
type InvalidationDateStrategy struct {
	Document  *invalidation_models.InvalidationDocument
	Deadlines *policy.Deadlines
	Now       time.Time
}

func (s *InvalidationDateStrategy) Validate() *dte_errors.DTEError {
	if s.Document.Identification == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Identification")
	}
	if s.Document.Document == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Document")
	}

	doc := s.Document.Document
	docType := doc.Type.GetValue()

	if doc.StampDate.IsZero() {
		return dte_errors.NewDTEErrorSimple("MissingStampDate", doc.GenerationCode.GetValue())
	}

	deadline := s.Deadlines.TransmissionDeadline(docType, doc.StampDate, doc.IssuerActivity)
	if s.Now.After(deadline) {
		return dte_errors.NewDTEErrorSimple("InvalidationDeadlineExpired", docType, deadline.Format(deadlineLayout))
	}

	return s.validateEventDate(docType, doc)
}

func (s *InvalidationDateStrategy) validateEventDate(docType string, doc *invalidation_models.InvalidatedDocument) *dte_errors.DTEError {
	if doc.GenerationDateTime.IsZero() {
		return nil
	}

	eventDate := s.Document.Identification.EmissionDate.GetValue()
	generation := doc.GenerationDateTime

	if policy.IsInvoiceType(docType) {
		maxEventDate := policy.AddValidityPeriod(docType, generation, doc.IssuerActivity)
		if eventDate.Format("2006-01-02") > maxEventDate.Format("2006-01-02") {
			return dte_errors.NewDTEErrorSimple("InvalidationEventDateOutOfRange", docType, maxEventDate.Format("2006-01-02"))
		}
		return nil
	}

	if !policy.SameDay(generation, eventDate) {
		return dte_errors.NewDTEErrorSimple("InvalidationEventDateMismatch", docType, generation.Format("2006-01-02"))
	}
	return nil
}
