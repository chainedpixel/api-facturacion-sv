package retention

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapRetentionResponseSummary maps the retention summary domain model to the response structure.
func MapRetentionResponseSummary(summary *retention_models.RetentionSummary) *structs.RetentionSummary {
	if summary == nil {
		return nil
	}

	return &structs.RetentionSummary{
		TotalSujRetencion: summary.TotalSubjectRetention.GetValue(),
		TotalIva:          summary.TotalIVA.GetValue(),
		TotalIvaRetenido:  summary.TotalIVARetention.GetValue(),
		TotalLetras:       summary.TotalIVARetentionLetters,
		Observaciones:     summary.Observations,
	}
}
