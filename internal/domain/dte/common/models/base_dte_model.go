package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator"
)

// DTEDocument is a struct that represents a DTE document, containing Identification, Issuer, Receiver, Items, Summary, Extension and Appendix
type DTEDocument struct {
	Identification   interfaces.Identification    `json:"identification"`
	Issuer           interfaces.Issuer            `json:"issuer"`
	Receiver         interfaces.Receiver          `json:"receiver"`
	Items            []interfaces.Item            `json:"items"`
	Summary          interfaces.Summary           `json:"summary"`
	Extension        interfaces.Extension         `json:"extension,omitempty"`
	Appendix         []interfaces.Appendix        `json:"appendix,omitempty"`
	RelatedDocuments []interfaces.RelatedDocument `json:"related_documents,omitempty"`
	OtherDocuments   []interfaces.OtherDocuments  `json:"other_documents,omitempty"`
	ThirdPartySale   interfaces.ThirdPartySale    `json:"third_party_sale,omitempty"`
}

func (d *DTEDocument) GetIdentification() interfaces.Identification {
	return d.Identification
}
func (d *DTEDocument) GetIssuer() interfaces.Issuer {
	return d.Issuer
}
func (d *DTEDocument) GetReceiver() interfaces.Receiver {
	return d.Receiver
}
func (d *DTEDocument) GetItems() []interfaces.Item {
	return d.Items
}
func (d *DTEDocument) GetSummary() interfaces.Summary {
	return d.Summary
}
func (d *DTEDocument) GetAppendix() []interfaces.Appendix {
	return d.Appendix
}
func (d *DTEDocument) GetExtension() interfaces.Extension {
	return d.Extension
}
func (d *DTEDocument) GetRelatedDocuments() []interfaces.RelatedDocument {
	return d.RelatedDocuments
}
func (d *DTEDocument) GetOtherDocuments() []interfaces.OtherDocuments {
	return d.OtherDocuments
}
func (d *DTEDocument) GetThirdPartySale() interfaces.ThirdPartySale {
	return d.ThirdPartySale
}

func (d *DTEDocument) SetIdentification(identification interfaces.Identification) error {
	if identification == nil {
		return dte_errors.NewValidationError("RequiredField", "Identification")
	}
	d.Identification = identification
	return nil
}

func (d *DTEDocument) SetAppendix(appendix []interfaces.Appendix) error {
	d.Appendix = appendix
	return nil
}

func (d *DTEDocument) SetExtension(extension interfaces.Extension) error {
	d.Extension = extension
	return nil
}

func (d *DTEDocument) SetIssuer(issuer interfaces.Issuer) error {
	if issuer == nil {
		return dte_errors.NewValidationError("RequiredField", "Issuer")
	}
	d.Issuer = issuer
	return nil
}

func (d *DTEDocument) SetReceiver(receiver interfaces.Receiver) error {
	if receiver == nil {
		return dte_errors.NewValidationError("RequiredField", "Receiver")
	}
	d.Receiver = receiver
	return nil
}

func (d *DTEDocument) SetItems(items []interfaces.Item) error {
	if items == nil || len(items) == 0 {
		return dte_errors.NewValidationError("RequiredField", "Items")
	}
	d.Items = items
	return nil
}

func (d *DTEDocument) SetSummary(summary interfaces.Summary) error {
	if summary == nil {
		return dte_errors.NewValidationError("RequiredField", "Summary")
	}
	d.Summary = summary
	return nil
}

func (d *DTEDocument) SetRelatedDocuments(relatedDocuments []interfaces.RelatedDocument) error {
	d.RelatedDocuments = relatedDocuments
	return nil
}

func (d *DTEDocument) SetOtherDocuments(otherDocuments []interfaces.OtherDocuments) error {
	d.OtherDocuments = otherDocuments
	return nil
}

func (d *DTEDocument) SetThirdPartySale(thirdPartySale interfaces.ThirdPartySale) error {
	d.ThirdPartySale = thirdPartySale
	return nil
}

// Validate Validates a DTE document against the validation rules
func (d *DTEDocument) Validate() error {
	return validator.ValidateDTEDocument(d)
}

// ValidateDTERules Validates the business rules of a DTE document and returns an error if they are not met
func (d *DTEDocument) ValidateDTERules() *dte_errors.DTEError {
	rulesValidator := validator.NewDTERulesValidator(d)
	return rulesValidator.Validate()
}
