package fse

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

func MapFSERequestSummary(summaryReq *structs.FSESummaryRequest) (*fse_models.FSESummary, error) {
	if summaryReq == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "Summary")
	}

	if summaryReq.TotalInWords == nil && summaryReq.TotalToPay != 0 {
		inLetters := utils.InLetters(summaryReq.TotalToPay)
		summaryReq.TotalInWords = &inLetters
	}

	baseSummary, err := common.MapCommonRequestSummary(structs.SummaryRequest{
		TotalOperation:     summaryReq.TotalPurchase,
		TotalToPay:         summaryReq.TotalToPay,
		TotalInWords:       summaryReq.TotalInWords,
		OperationCondition: summaryReq.OperationCondition,
		PaymentTypes:       summaryReq.PaymentTypes,
		TotalDiscount:      summaryReq.TotalDiscount,
		SubTotal:           summaryReq.SubTotal,
		SubTotalSales:      summaryReq.TotalPurchase,
		NonSubjectDiscount: summaryReq.NonSubjectDiscount,
	})
	if err != nil {
		return nil, fmt.Errorf("error mapping base summary: %w", err)
	}

	totalPurchase, err := financial.NewAmount(summaryReq.TotalPurchase)
	if err != nil {
		return nil, fmt.Errorf("error creating total purchase amount: %w", err)
	}

	ivaRetention, err := financial.NewAmount(summaryReq.IVARetention)
	if err != nil {
		return nil, fmt.Errorf("error creating IVA retention amount: %w", err)
	}

	incomeRetention, err := financial.NewAmount(summaryReq.IncomeRetention)
	if err != nil {
		return nil, fmt.Errorf("error creating income retention amount: %w", err)
	}

	if summaryReq.TotalPurchase < 0 {
		return nil, dte_errors.NewValidationError("InvalidField", "Summary->TotalPurchase")
	}

	if summaryReq.IVARetention < 0 {
		return nil, dte_errors.NewValidationError("InvalidField", "Summary->IVARetention")
	}

	if summaryReq.IncomeRetention < 0 {
		return nil, dte_errors.NewValidationError("InvalidField", "Summary->IncomeRetention")
	}

	observations := summaryReq.Observations

	return fse_models.NewFSESummary(
		baseSummary,
		*totalPurchase,
		*ivaRetention,
		*incomeRetention,
		observations,
	), nil
}
