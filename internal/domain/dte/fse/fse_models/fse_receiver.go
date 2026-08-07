package fse_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
)

type FSEReceiver struct {
	*models.Receiver
	DocumentType        document.DTEType
	DocumentNumber      identification.DocumentNumber
	ActivityCode        *identification.ActivityCode
	ActivityDescription *string
}

func NewFSEReceiver(
	baseReceiver *models.Receiver,
	documentType document.DTEType,
	documentNumber identification.DocumentNumber,
	activityCode *identification.ActivityCode,
	activityDescription *string,
) *FSEReceiver {
	return &FSEReceiver{
		Receiver:            baseReceiver,
		DocumentType:        documentType,
		DocumentNumber:      documentNumber,
		ActivityCode:        activityCode,
		ActivityDescription: activityDescription,
	}
}

func (r *FSEReceiver) GetDocumentType() document.DTEType {
	return r.DocumentType
}

func (r *FSEReceiver) GetDocumentNumber() identification.DocumentNumber {
	return r.DocumentNumber
}

func (r *FSEReceiver) GetActivityCode() *identification.ActivityCode {
	return r.ActivityCode
}

func (r *FSEReceiver) GetActivityDescription() *string {
	return r.ActivityDescription
}

func (r *FSEReceiver) SetDocumentType(docType document.DTEType) {
	r.DocumentType = docType
}

func (r *FSEReceiver) SetDocumentNumber(docNumber identification.DocumentNumber) {
	r.DocumentNumber = docNumber
}

func (r *FSEReceiver) SetActivityCode(actCode *identification.ActivityCode) {
	r.ActivityCode = actCode
}

func (r *FSEReceiver) SetActivityDescription(actDesc *string) {
	r.ActivityDescription = actDesc
}
