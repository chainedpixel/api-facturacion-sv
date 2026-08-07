package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// MapCommonRequestOtherDocuments maps a list of additional documents to an additional document model -> Source: Request
func MapCommonRequestOtherDocuments(otherDocuments []structs.OtherDocRequest) ([]models.OtherDocument, error) {
	otherDocs := make([]models.OtherDocument, len(otherDocuments))

	for i, doc := range otherDocuments {
		mappedDocument, err := MapCommonRequestOtherDocument(doc)
		if err != nil {
			return nil, err
		}
		otherDocs[i] = *mappedDocument
	}

	return otherDocs, nil
}

// MapCommonRequestOtherDocument maps an additional document to an additional document model -> Source: Request
func MapCommonRequestOtherDocument(doc structs.OtherDocRequest) (*models.OtherDocument, error) {
	var doctor *models.DoctorInfo
	var err error

	if doc.DocumentCode == 3 && doc.Doctor == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "OtherDocuments->Doctor, when DocumentCode is 3")
	}

	documentCode, err := document.NewAssociatedDocumentCode(doc.DocumentCode)
	if err != nil {
		return nil, err
	}

	if doc.Doctor != nil {
		doctor, err = MapCommonRequestDoctorInfo(*doc.Doctor)
		if err != nil {
			return nil, err
		}
	}

	return &models.OtherDocument{
		AssociatedCode: *documentCode,
		Description:    doc.Description,
		Detail:         doc.Detail,
		Doctor:         doctor,
	}, nil
}
