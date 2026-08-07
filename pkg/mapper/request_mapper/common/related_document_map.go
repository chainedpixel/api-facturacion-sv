package common

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// MapCommonRequestRelatedDocuments maps a list of related documents to a related document model -> Source: Request
func MapCommonRequestRelatedDocuments(relatedDocuments []structs.RelatedDocRequest) ([]models.RelatedDocument, error) {
	result := make([]models.RelatedDocument, len(relatedDocuments))

	for i, relatedDocument := range relatedDocuments {
		relatedDoc, err := MapCommonRequestRelatedDocumentIndex(relatedDocument)
		if err != nil {
			return nil, err
		}
		result[i] = *relatedDoc
	}

	return result, nil
}

// MapCommonRequestRelatedDocumentIndex maps a related document to a related document model -> Source: Request
func MapCommonRequestRelatedDocumentIndex(doc structs.RelatedDocRequest) (*models.RelatedDocument, error) {

	dteType, err := document.NewDTEType(doc.DocumentType)
	if err != nil {
		return nil, err
	}

	generationType, err := document.NewModelType(doc.GenerationType)
	if err != nil {
		return nil, err
	}

	docNumber, err := document.NewDocumentNumber(doc.DocumentNumber, doc.GenerationType)
	if err != nil {
		return nil, err
	}

	if doc.EmissionDate == "" {
		doc.EmissionDate = utils.TimeNow().Format("2006-01-02")
	}

	timeParse, err := time.Parse("2006-01-02", doc.EmissionDate)
	if err != nil {
		return nil, err
	}

	emissionDate, err := temporal.NewEmissionDate(timeParse)
	if err != nil {
		return nil, err
	}

	return &models.RelatedDocument{
		DocumentType:   *dteType,
		GenerationType: *generationType,
		DocumentNumber: docNumber.GetValue(),
		EmissionDate:   *emissionDate,
	}, nil
}
