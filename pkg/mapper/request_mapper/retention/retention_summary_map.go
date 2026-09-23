package retention

import (
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

func MapRetentionSummary(req *structs.RetentionSummary) (*retention_models.RetentionSummary, error) {
	if req == nil {
		return nil, dte_errors.NewValidationError("RequiredField", "RetentionSummary")
	}

	if req.TotalRetentionAmount == 0 {
		return nil, dte_errors.NewValidationError("RequiredField", "RetentionSummary->TotalRetentionAmount")
	}

	if req.TotalRetentionIVA == 0 {
		return nil, dte_errors.NewValidationError("RequiredField", "RetentionSummary->TotalRetentionIVA")
	}

	totalRetention, err := financial.NewAmountForTotal(req.TotalRetentionAmount)
	if err != nil {
		return nil, err
	}

	totalIVARetention, err := financial.NewAmountForTotal(req.TotalRetentionIVA)
	if err != nil {
		return nil, err
	}

	ivaVal := req.TotalIVA
	if ivaVal == 0 {
		ivaVal = math.Round(req.TotalRetentionAmount*0.13*100) / 100
	}
	totalIVA, err := financial.NewAmountForTotal(ivaVal)
	if err != nil {
		return nil, err
	}

	if req.Observations != nil && len(*req.Observations) > 3000 {
		return nil, dte_errors.NewValidationError("InvalidLength", "RetentionSummary->Observations", 3000)
	}

	return &retention_models.RetentionSummary{
		TotalSubjectRetention: *totalRetention,
		TotalIVA:              *totalIVA,
		TotalIVARetention:     *totalIVARetention,
		Observations:          req.Observations,
	}, nil
}
