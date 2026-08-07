package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
)

type RemissionNoteReceiverStrategy struct {
	Document *remission_note_models.RemissionNoteModel
}

func (s *RemissionNoteReceiverStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return dte_errors.NewDTEErrorSimple("RemissionNoteReceiverValidationFailed")
	}

	if s.Document.Receiver == nil {
		return dte_errors.NewDTEErrorSimple("RequiredField", "receiver")
	}

	remissionReceiver, ok := s.Document.Receiver.(*remission_note_models.RemissionNoteReceiver)
	if !ok || remissionReceiver == nil {
		return dte_errors.NewDTEErrorSimple("InvalidRemissionNoteReceiver")
	}

	var errors []*dte_errors.DTEError

	if err := s.validateBienTitulo(remissionReceiver); err != nil {
		errors = append(errors, err)
	}

	if err := s.validateBaseReceiverFields(remissionReceiver); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return dte_errors.NewDTEErrorComposite(errors)
	}

	return nil
}

func (s *RemissionNoteReceiverStrategy) validateBienTitulo(receiver *remission_note_models.RemissionNoteReceiver) *dte_errors.DTEError {
	bienTitulo := receiver.GetBienTitulo()

	if bienTitulo == nil || *bienTitulo == "" {
		return dte_errors.NewDTEErrorSimple("RequiredBienTitulo")
	}

	if len(*bienTitulo) > 2 {
		return dte_errors.NewDTEErrorSimple("InvalidLength", "bienTitulo", "2", *bienTitulo)
	}

	validValues := map[string]bool{
		"01": true,
		"02": true,
		"03": true,
		"04": true,
		"05": true,
		"99": true,
	}

	if !validValues[*bienTitulo] {
		return dte_errors.NewDTEErrorSimple("InvalidBienTitulo", *bienTitulo)
	}

	return nil
}

func (s *RemissionNoteReceiverStrategy) validateBaseReceiverFields(receiver *remission_note_models.RemissionNoteReceiver) *dte_errors.DTEError {
	var errors []*dte_errors.DTEError

	if receiver.GetName() == nil || *receiver.GetName() == "" {
		errors = append(errors, dte_errors.NewDTEErrorSimple("RequiredField", "name"))
	}

	if receiver.GetDocumentType() == nil || *receiver.GetDocumentType() == "" {
		errors = append(errors, dte_errors.NewDTEErrorSimple("RequiredField", "document_type"))
	}

	if receiver.GetDocumentNumber() == nil || *receiver.GetDocumentNumber() == "" {
		errors = append(errors, dte_errors.NewDTEErrorSimple("RequiredField", "document_number"))
	}

	if receiver.GetAddress() == nil {
		errors = append(errors, dte_errors.NewDTEErrorSimple("RequiredField", "address"))
	}

	if receiver.GetEmail() == nil || *receiver.GetEmail() == "" {
		errors = append(errors, dte_errors.NewDTEErrorSimple("RequiredField", "email"))
	}

	if docType := receiver.GetDocumentType(); docType != nil && *docType == "36" {
		if receiver.GetNIT() == nil {
			errors = append(errors, dte_errors.NewDTEErrorSimple("RequiredField", "nit"))
		}
	}

	if len(errors) > 0 {
		return dte_errors.NewDTEErrorComposite(errors)
	}

	return nil
}
