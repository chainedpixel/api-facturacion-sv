package ccf_models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type CreditFiscalDocument struct {
	*models.DTEDocument
	CreditItems   []CreditItem
	CreditSummary CreditSummary
}
