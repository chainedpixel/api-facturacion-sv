package retention

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapRetentionResponseSummary(summary *retention_models.RetentionSummary) *structs.RetentionSummary {
	if summary == nil {
		return nil
	}

	return &structs.RetentionSummary{
		TotalIvaRetenido:       summary.TotalIVARetention.GetValue(),
		TotalSujRetencion:      summary.TotalSubjectRetention.GetValue(),
		TotalIvaRetenidoLetras: summary.TotalIVARetentionLetters,
	}
}
