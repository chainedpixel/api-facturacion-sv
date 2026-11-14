package retention

import (
	"math"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// SingleRetentionCode devuelve el codigo de retencion (CAT-006) comun a todos los items.
// Hacienda rechaza un CRE con codigos distintos ("NO SE PERMITEN VALORES DIFERENTES").
func SingleRetentionCode(items []structs.RetentionItem) (string, error) {
	code := ""
	for _, item := range items {
		if code == "" {
			code = item.RetentionCode
			continue
		}
		if item.RetentionCode != code {
			return "", dte_errors.NewValidationError("MixedRetentionCodes", code, item.RetentionCode)
		}
	}
	return code, nil
}

// AllElectronic indica si todos los documentos informados en el CRE son electronicos (tipo de generacion 2)
func AllElectronic(items []structs.RetentionItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.DocumentType != constants.ElectronicDocument {
			return false
		}
	}
	return true
}

// MapRetentionSummary mapea el resumen del CRE. El IVA 13% (campo 157) que no envia el cliente se calcula con
// retention_models.ExpectedTotalIVA: solo aplica a documentos electronicos con codigo C4.
func MapRetentionSummary(req *structs.RetentionSummary, retentionCode string, allElectronic bool) (*retention_models.RetentionSummary, error) {
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
		ivaVal = math.Round(retention_models.ExpectedTotalIVA(req.TotalRetentionAmount, retentionCode, allElectronic)*100) / 100
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
