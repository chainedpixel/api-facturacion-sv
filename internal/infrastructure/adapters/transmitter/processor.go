package transmitter

import (
	models2 "github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
)

type DocumentProcessor interface {
	ProcessRequest(signedDoc string, document interface{}) (*models2.HaciendaRequest, error)
	ProcessResponse(resp *models2.HaciendaResponse) (*models2.TransmitResult, error)
}
