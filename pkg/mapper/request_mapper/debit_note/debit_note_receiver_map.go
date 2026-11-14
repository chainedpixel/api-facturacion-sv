package debit_note

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// MapDebitNoteRequestReceiver mapea el receptor; NCE y NDE comparten las mismas reglas (ver common.MapAdjustmentRequestReceiver)
func MapDebitNoteRequestReceiver(receiver *structs.ReceiverRequest) (*models.Receiver, error) {
	return common.MapAdjustmentRequestReceiver(receiver)
}
