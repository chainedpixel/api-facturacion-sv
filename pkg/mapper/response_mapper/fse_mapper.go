package response_mapper

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/fse"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// ToMHFSE converts an FSE domain model to the format required by the Ministry of Finance
func ToMHFSE(doc interface{}) *structs.FSEDTEResponse {
	fseDoc := doc.(*fse_models.FSEModel)

	fseResponse := &structs.FSEDTEResponse{
		Identificacion:  *common.MapCommonResponseIdentification(fseDoc.Identification),
		Emisor:          fse.MapFSEResponseIssuer(fseDoc.Issuer),
		Receptor:        fse.MapFSEResponseReceiver(fseDoc.FSEReceiver),
		Resumen:         fse.MapFSEResponseSummary(fseDoc.FSESummary),
		CuerpoDocumento: fse.MapFSEResponseItems(fseDoc.FSEItems),
	}

	if len(fseDoc.GetAppendix()) > 0 {
		appendices := common.MapCommonResponseAppendix(fseDoc.GetAppendix())
		fseResponse.Apendice = &appendices
	} else {
		fseResponse.Apendice = nil
	}

	return fseResponse
}
