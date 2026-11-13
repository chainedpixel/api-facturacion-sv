package common

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

// AdjustsRetentionVoucher indica si una NCE/NDE ajusta un Comprobante de Retencion (07).
// Anexo IV, campos 100 y 146: el "IVA 13%" por item y su total aplican al ajustar un CRE; cuando se
// ajusta un CCFE el IVA se informa unicamente como tributo 20 del resumen y totalIva va en 0.00
// (Hacienda rechaza un totalIva distinto de 0 con "CALCULO INCORRECTO" / "excede el maximo permitido").
func AdjustsRetentionVoucher(related []interfaces.RelatedDocument) bool {
	for _, doc := range related {
		if doc.GetDocumentType() == constants.ComprobanteRetencionElectronico {
			return true
		}
	}
	return false
}

// AdjustmentItemIVA calcula el IVA percibido o retenido de un item de NCE/NDE (Anexo IV, campos 99 y 101):
// ventaGravada x 1% cuando el resumen informa percepcion o retencion; el resumen es la suma de los items.
func AdjustmentItemIVA(taxedSale float64, summaryAmount float64) float64 {
	if summaryAmount <= 0 || taxedSale <= 0 {
		return 0
	}
	return decimal.NewFromFloat(taxedSale).Mul(decimal.RequireFromString("0.01")).Round(8).InexactFloat64()
}
