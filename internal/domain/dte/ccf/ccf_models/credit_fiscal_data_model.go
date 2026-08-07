package ccf_models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type CCFData struct {
	*models.InputDataCommon
	Items         []CreditItem
	CreditSummary *CreditSummary
}
