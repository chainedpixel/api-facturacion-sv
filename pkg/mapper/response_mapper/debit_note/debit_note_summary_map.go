package debit_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapDebitNoteResponseSummary(summary debit_note_models.DebitNoteSummary) *structs.DebitNoteDTESummary {
	return &structs.DebitNoteDTESummary{
		TotalNoSuj:          summary.GetTotalNonSubject(),
		TotalExenta:         summary.GetTotalExempt(),
		TotalGravada:        summary.GetTotalTaxed(),
		SubTotalVentas:      summary.GetSubTotal(),
		DescuNoSuj:          summary.GetNonSubjectDiscount(),
		DescuExenta:         summary.GetExemptDiscount(),
		DescuGravada:        summary.TaxedDiscount.GetValue(),
		TotalDescu:          summary.GetTotalDiscount(),
		SubTotal:            summary.GetSubTotal(),
		Tributos:            common.MapTaxes(summary.GetTotalTaxes()),
		IvaRete1:            summary.IVARetention.GetValue(),
		IvaPerci1:           summary.IVAPerception.GetValue(),
		ReteRenta:           summary.IncomeRetention.GetValue(),
		MontoTotalOperacion: summary.GetTotalOperation(),
		TotalLetras:         summary.GetTotalInWords(),
		CondicionOperacion:  summary.GetOperationCondition(),
		NumPagoElectronico:  summary.GetElectronicPayment(),
	}
}
