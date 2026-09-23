package credit_note

import (
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCreditNoteResponseSummary maps a credit note domain summary to the Ministry of Finance format.
func MapCreditNoteResponseSummary(summary credit_note_models.CreditNoteSummary) *structs.CreditNoteDTESummary {
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

	return &structs.CreditNoteDTESummary{
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
		Observaciones:       summary.GetObservations(),
		CodigoRetencionMH:   nil,
	}
}
