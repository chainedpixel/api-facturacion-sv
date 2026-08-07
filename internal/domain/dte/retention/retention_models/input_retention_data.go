package retention_models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type InputRetentionData struct {
	*models.InputDataCommon
	RetentionItems   []RetentionItem   `json:"retention_items"`
	RetentionSummary *RetentionSummary `json:"retention_summary,omitempty"`
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
