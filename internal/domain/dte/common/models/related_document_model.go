package models

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
)

// RelatedDocument represents a document related to the invoice
type RelatedDocument struct {
	DocumentType   document.DTEType      `json:"documentType"`
	GenerationType document.ModelType    `json:"generationType"`
	DocumentNumber string                `json:"documentNumber"`
	EmissionDate   temporal.EmissionDate `json:"emissionDate"`
}

func (r *RelatedDocument) GetDocumentType() string {
	return r.DocumentType.GetValue()
}

func (r *RelatedDocument) GetGenerationType() int {
	return r.GenerationType.GetValue()
}

func (r *RelatedDocument) GetDocumentNumber() string {
	return r.DocumentNumber
}

func (r *RelatedDocument) GetEmissionDate() time.Time {
	return r.EmissionDate.GetValue()
}
func (r *RelatedDocument) SetDocumentType(documentType string) error {
	dtObj, err := document.NewDTEType(documentType)
	if err != nil {
		return err
	}
	r.DocumentType = *dtObj
	return nil
}

func (r *RelatedDocument) SetGenerationType(generationType int) error {
	gtObj, err := document.NewModelType(generationType)
	if err != nil {
		return err
	}
	r.GenerationType = *gtObj
	return nil
}

func (r *RelatedDocument) SetDocumentNumber(documentNumber string) error {
	if documentNumber == "" {
		return dte_errors.NewValidationError("RequiredField", "DocumentNumber")
	}
	r.DocumentNumber = documentNumber
	return nil
}

func (r *RelatedDocument) SetEmissionDate(emissionDate time.Time) error {
	edObj, err := temporal.NewEmissionDate(emissionDate)
	if err != nil {
		return err
	}
	r.EmissionDate = *edObj
	return nil
}
