package common

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
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
		// Los tributos del resumen se expresan con 2 decimales (Regla 7.2 y multipleOf 0.01 en los schemas)
		taxAmount, err := financial.NewAmountForTotal(tax.Value)
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
