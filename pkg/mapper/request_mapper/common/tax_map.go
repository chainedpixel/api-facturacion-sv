package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// MapCommonRequestSummaryTaxes converts an array of TaxRequest to an array of Tax, but of SummaryTax, not items
func MapCommonRequestSummaryTaxes(taxes []structs.TaxRequest) ([]interfaces.Tax, error) {
	result := make([]interfaces.Tax, len(taxes))
	for i, tax := range taxes {
		if tax.Code == "" || tax.Description == "" {
			return nil, shared_error.NewFormattedGeneralServiceError("CommonMapper", "MapCommonRequestSummaryTaxes", "InvalidTaxCodeAndDescription")
		}

		if tax.Value == 0 && tax.Code != constants.TaxIVAExport {
			return nil, dte_errors.NewValidationError("RequiredField", "Summary->Taxes->Value")
		}

		taxVO, err := financial.NewTaxType(tax.Code)
		if err != nil {
			return nil, err
		}
		taxAmount, err := financial.NewAmount(tax.Value)
		if err != nil {
			return nil, err
		}

		result[i] = &models.Tax{
			Code:        *taxVO,
			Value:       &models.TaxAmount{TotalAmount: *taxAmount},
			Description: tax.Description,
		}
	}

	return result, nil
}
