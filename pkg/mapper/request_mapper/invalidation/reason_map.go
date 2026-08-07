package invalidation

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapInvalidationReasonRequest(reason *structs.ReasonRequest) (*invalidation_models.InvalidationReason, error) {
	invalidationType, err := document.NewInvalidationType(reason.Type)
	if err != nil {
		return nil, err
	}

	responsibleDocType, err := document.NewDTETypeForReceiver(reason.ResponsibleDocType)
	if err != nil {
		return nil, err
	}

	requestorDocType, err := document.NewDTETypeForReceiver(reason.RequestorDocType)
	if err != nil {
		return nil, err
	}

	responsibleName := reason.ResponsibleName
	requestorName := reason.RequestorName

	responsibleDocNum, err := identification.NewDocumentNumber(reason.ResponsibleNumDoc, reason.ResponsibleDocType)
	if err != nil {
		return nil, err
	}

	requestorDocNum, err := identification.NewDocumentNumber(reason.RequestorNumDoc, reason.RequestorDocType)
	if err != nil {
		return nil, err
	}

	result := &invalidation_models.InvalidationReason{
		Type:               *invalidationType,
		ResponsibleName:    responsibleName,
		ResponsibleDocType: *responsibleDocType,
		ResponsibleDocNum:  *responsibleDocNum,
		RequesterName:      requestorName,
		RequesterDocType:   *requestorDocType,
		RequesterDocNum:    *requestorDocNum,
	}

	if reason.Reason != nil && reason.Type == 3 {
		invalidationReason, err := document.NewInvalidationReason(*reason.Reason)
		if err != nil {
			return nil, err
		}
		result.Reason = invalidationReason
	}

	return result, nil
}
