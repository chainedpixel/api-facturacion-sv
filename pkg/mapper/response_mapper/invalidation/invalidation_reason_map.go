package invalidation

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/structs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
)

func MapInvalidationReasonResponse(reason *invalidation_models.InvalidationReason) *structs.ReasonResponse {
	if reason == nil {
		return nil
	}

	result := &structs.ReasonResponse{
		TipoAnulacion:     reason.Type.GetValue(),
		NombreResponsable: reason.ResponsibleName,
		TipDocResponsable: reason.ResponsibleDocType.GetValue(),
		NumDocResponsable: reason.ResponsibleDocNum.GetValue(),
		NombreSolicita:    reason.RequesterName,
		TipDocSolicita:    reason.RequesterDocType.GetValue(),
		NumDocSolicita:    reason.RequesterDocNum.GetValue(),
	}

	if reason.Reason != nil {
		result.MotivoAnulacion = utils.ToStringPointer(reason.Reason.GetValue())
	}

	return result
}
