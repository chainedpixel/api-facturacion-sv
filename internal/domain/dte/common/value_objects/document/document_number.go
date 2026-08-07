package document

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

const (
	PhysicalFormat   = "^[a-zA-Z0-9]{1,20}$"
	ElectronicFormat = "^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$"
)

type DocumentNumber struct {
	Number       string
	DocumentType int
}

func NewDocumentNumber(number string, documentType int) (DocumentNumber, error) {
	value := DocumentNumber{Number: number, DocumentType: documentType}
	if value.IsValid() {
		return value, nil
	}

	return DocumentNumber{}, dte_errors.NewValidationError("InvalidDocumentNumberItem", number)
}

func (d *DocumentNumber) NewValidatedDocumentNumber(number string, documentType int) (DocumentNumber, error) {
	value := DocumentNumber{Number: number, DocumentType: documentType}
	if value.IsValid() {
		return value, nil
	}

	return DocumentNumber{}, dte_errors.NewValidationError("InvalidDocumentNumberItem", number)
}

func (d *DocumentNumber) GetValue() string {
	return d.Number
}

func (d *DocumentNumber) IsValid() bool {
	if d.DocumentType == 1 {
		matchString, _ := regexp.MatchString(PhysicalFormat, d.Number)
		return matchString
	}

	matchString, _ := regexp.MatchString(ElectronicFormat, d.Number)
	return matchString
}

func (d *DocumentNumber) Equals(other interfaces.ValueObject[string]) bool {
	return d.GetValue() == other.GetValue()
}

func (d *DocumentNumber) ToString() string {
	return d.Number
}
