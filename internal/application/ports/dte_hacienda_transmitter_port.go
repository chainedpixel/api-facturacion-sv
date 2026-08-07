package ports

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
)

// DTETransmitter defines the behavior of an electronic document transmitter
type DTETransmitter interface {
	Transmit(context.Context, interface{}, string, string) (*models.TransmitResult, error)
	CheckDocumentStatus(context.Context, interface{}, string) (*models.TransmitResult, error)
	SendToHacienda(context.Context, *models.HaciendaRequest, string) (*models.HaciendaResponse, error)
}
