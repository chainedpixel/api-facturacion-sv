package strategy

import (
	"regexp"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

const (
	DUIPattern = `^[0-9]{8}-[0-9]{1}$`
	NITPattern = `^([0-9]{14}|[0-9]{9})$`
)

var (
	nitRegex = regexp.MustCompile(NITPattern)
	duiRegex = regexp.MustCompile(DUIPattern)
)

type BasicRulesStrategy struct {
	Document interfaces.DTEDocument
}

// Validate Validates the basic rules of a DTE document
func (s *BasicRulesStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return dte_errors.NewDTEErrorSimple("RequiredFieldMissing", "Document", "nil")
	}

	if s.Document.GetIdentification() == nil {
		return dte_errors.NewDTEErrorSimple("RequiredFieldMissing", "Identification", "nil")
	}

	docType := s.Document.GetIdentification().GetDTEType()

	if s.Document.GetIssuer() == nil {
		return dte_errors.NewDTEErrorSimple("RequiredFieldMissing", "Issuer", docType)
	}

	if s.Document.GetItems() == nil || len(s.Document.GetItems()) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredFieldMissing", "Items", docType)
	}

	if requiresReceiver(docType) {
		receiver := s.Document.GetReceiver()
		if receiver == nil {
			return dte_errors.NewDTEErrorSimple("RequiredFieldMissing", "Receiver", docType)
		}
	}

	err := getDocumentNumberError(s.Document.GetReceiver().GetDocumentNumber(), s.Document.GetReceiver().GetDocumentType())
	if err != nil {
		return err
	}

	return nil
}

// requiresReceiver Checks whether the document type requires a receiver
func requiresReceiver(docType string) bool {
	switch docType {
	case constants.FacturaElectronica,
		constants.CCFElectronico,
		constants.NotaRemisionElectronica,
		constants.NotaCreditoElectronica,
		constants.NotaDebitoElectronica,
		constants.FacturaExportacionElectronica:
		return true
	default:
		return false
	}
}

func getDocumentNumberError(documentNumber, documentType *string) *dte_errors.DTEError {
	if documentNumber == nil || documentType == nil {
		return nil
	}

	switch *documentType {
	case constants.NIT:
		if !nitRegex.MatchString(*documentNumber) {
			return dte_errors.NewDTEErrorSimple("InvalidNITFormat", *documentNumber)
		}
	case constants.DUI:
		if !duiRegex.MatchString(*documentNumber) {
			return dte_errors.NewDTEErrorSimple("InvalidDUIFormat", *documentNumber)
		}
	}

	return nil
}
