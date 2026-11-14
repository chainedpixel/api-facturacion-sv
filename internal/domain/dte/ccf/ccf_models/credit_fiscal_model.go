package ccf_models

import "github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"

type CreditFiscalDocument struct {
	*models.DTEDocument
	CreditItems   []CreditItem
	CreditSummary CreditSummary
}
