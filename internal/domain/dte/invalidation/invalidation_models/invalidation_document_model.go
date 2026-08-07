package invalidation_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
)

type InvalidationDocument struct {
	Identification *models.Identification `json:"identification,omitempty"`
	Issuer         *models.Issuer         `json:"issuer,omitempty"`
	Document       *InvalidatedDocument   `json:"document,omitempty"`
	Reason         *InvalidationReason    `json:"reason,omitempty"`
}
