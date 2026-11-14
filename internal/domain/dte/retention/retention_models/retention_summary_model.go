package retention_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/financial"

// RetentionSummary represents the summary of a retention document.
type RetentionSummary struct {
	TotalSubjectRetention    financial.Amount
	TotalIVA                 financial.Amount
	TotalIVARetention        financial.Amount
	TotalIVARetentionLetters string
	Observations             *string
}
