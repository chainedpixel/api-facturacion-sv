package retention_models

import "github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/common/models"

type InputRetentionData struct {
	*models.InputDataCommon
	RetentionItems   []RetentionItem   `json:"retention_items"`             // Lista de items de retención
	RetentionSummary *RetentionSummary `json:"retention_summary,omitempty"` // Resumen de la retención
}

func (i *InputRetentionData) IsAllPhysical() bool {
	isAllPhysical := true
	for _, item := range i.RetentionItems {
		if item.DocumentType.GetValue() == 2 {
			isAllPhysical = false
		}
	}
	return isAllPhysical
}
