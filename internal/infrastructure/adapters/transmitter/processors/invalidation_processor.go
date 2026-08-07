package processors

import (
	"github.com/chainedpixel/ordo-factus/config"
	models2 "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type InvalidationProcessor struct{}

func (p *InvalidationProcessor) ProcessRequest(signedDoc string, document interface{}) (*models2.HaciendaRequest, error) {
	version, dteType, generationCode, sequenceNumber, err := GetDocumentRequestData(document)
	if err != nil {
		logs.Error("Failed to get document request data", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return &models2.HaciendaRequest{
		Ambient:        config.Server.AmbientCode,
		SendID:         sequenceNumber,
		Version:        version,
		Document:       signedDoc,
		DTEType:        dteType,
		GenerationCode: generationCode,
		URL:            config.MHPaths.NullifyURL,
	}, nil
}

func (p *InvalidationProcessor) ProcessResponse(resp *models2.HaciendaResponse) (*models2.TransmitResult, error) {
	if resp == nil {
		return nil, shared_error.NewGeneralServiceError("InvalidationProcessor", "ProcessResponse", "nil response", nil)
	}

	return &models2.TransmitResult{
		Status:         resp.Status,
		ReceptionStamp: &resp.ReceptionStamp,
		ProcessingDate: resp.ProcessingDate,
		MessageCode:    resp.MessageCode,
		MessageDesc:    resp.DescriptionMessage,
		Observations:   resp.Observations,
	}, nil
}
