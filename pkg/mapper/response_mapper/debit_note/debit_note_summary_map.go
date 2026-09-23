package debit_note

import (
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapDebitNoteResponseSummary maps a debit note domain summary to the Ministry of Finance format.
func MapDebitNoteResponseSummary(summary debit_note_models.DebitNoteSummary) *structs.DebitNoteDTESummary {
	var totalIva float64
	for _, tax := range summary.GetTotalTaxes() {
		if tax.GetCode() == "20" {
			totalIva = tax.GetValue()
			break
		}
	}
	if totalIva == 0 && summary.GetTotalTaxed() > 0 {
		totalIva = math.Round(summary.GetTotalTaxed()*0.13*100) / 100
	}

	return &structs.DebitNoteDTESummary{
		TotalNoSuj:          summary.GetTotalNonSubject(),
		TotalExenta:         summary.GetTotalExempt(),
		TotalGravada:        summary.GetTotalTaxed(),
		SubTotalVentas:      summary.GetSubtotalSales(),
		TotalDescu:          summary.GetTotalDiscount(),
		Tributos:            common.MapTaxes(summary.GetTotalTaxes()),
		MontoTotalOperacion: summary.GetTotalOperation(),
		IvaPerci:            summary.IVAPerception.GetValue(),
		TotalIva:            totalIva,
		IvaRete:             summary.IVARetention.GetValue(),
		TotalNoGravado:      summary.GetTotalNotTaxed(),
		TotalPagar:          summary.GetTotalToPay(),
		TotalLetras:         summary.GetTotalInWords(),
		CondicionOperacion:  summary.GetOperationCondition(),
		NumPagoElectronico:  summary.GetElectronicPayment(),
		Observaciones:       summary.GetObservations(),
		CodigoRetencionMH:   nil,
	}
}
