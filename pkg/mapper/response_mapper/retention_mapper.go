package response_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/retention"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func ToMHRetention(doc interface{}) *structs.RetentionDTEResponse {

	cast := doc.(*retention_models.RetentionModel)
	dte := &structs.RetentionDTEResponse{
		Identificacion:  common.MapCommonResponseIdentification(cast.Identification),
		Emisor:          retention.MapRetentionResponseIssuer(cast.Issuer),
		Receptor:        common.MapCommonResponseReceiver(cast.Receiver),
		Resumen:         retention.MapRetentionResponseSummary(cast.RetentionSummary),
		CuerpoDocumento: retention.MapRetentionResponseItem(cast.RetentionItems),
		Extension:       retention.MapRetentionResponseExtension(cast.Extension),
	}

	if cast.Appendix != nil {
		dte.Apendice = common.MapCommonResponseAppendix(cast.Appendix)
	}

	return dte
}
