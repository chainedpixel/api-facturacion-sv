package debit_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

func MapDebitNoteRequestSummary(summary *structs.DebitNoteSummaryRequest) (*debit_note_models.DebitNoteSummary, error) {
	if summary.TotalInWords == nil {
		inLetters := utils.InLetters(summary.TotalOperation)
		summary.TotalInWords = &inLetters
	}

	baseSummary, err := common.MapCommonRequestSummary(structs.SummaryRequest{
		TotalNonSubject:    summary.TotalNonSubject,
		TotalExempt:        summary.TotalExempt,
		TotalTaxed:         summary.TotalTaxed,
		SubTotal:           summary.SubTotal,
		NonSubjectDiscount: summary.NonSubjectDiscount,
		ExemptDiscount:     summary.ExemptDiscount,
		DiscountPercentage: summary.DiscountPercentage,
		TotalDiscount:      summary.TotalDiscount,
		TotalOperation:     summary.TotalOperation,
		TotalNonTaxed:      summary.TotalNonTaxed,
		SubTotalSales:      summary.SubTotalSales,
		TotalToPay:         summary.TotalToPay,
		OperationCondition: summary.OperationCondition,
		Taxes:              summary.Taxes,
		PaymentTypes:       []structs.PaymentRequest{},
		TotalInWords:       summary.TotalInWords,
	})

	if err != nil {
		return nil, err
	}

	taxedDiscount, err := financial.NewAmountForTotal(summary.TaxedDiscount)
	if err != nil {
		return nil, err
	}

	ivaPerception, err := financial.NewAmountForTotal(summary.IVAPerception)
	if err != nil {
		return nil, err
	}

	ivaRetention, err := financial.NewAmountForTotal(summary.IVARetention)
	if err != nil {
		return nil, err
	}

	incomeRetention, err := financial.NewAmountForTotal(summary.IncomeRetention)
	if err != nil {
		return nil, err
	}

	return &debit_note_models.DebitNoteSummary{
		Summary:         baseSummary,
		TaxedDiscount:   *taxedDiscount,
		IVAPerception:   *ivaPerception,
		IVARetention:    *ivaRetention,
		IncomeRetention: *incomeRetention,
	}, nil
}
