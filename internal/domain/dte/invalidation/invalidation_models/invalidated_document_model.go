package invalidation_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
)

type InvalidatedDocument struct {
	Type            document.DTEType               `json:"type"`
	GenerationCode  identification.GenerationCode  `json:"generationCode"`
	ReceptionStamp  string                         `json:"receptionStamp"`
	ControlNumber   identification.ControlNumber   `json:"controlNumber"`
	EmissionDate    temporal.EmissionDate          `json:"emissionDate"`
	DocumentType    *document.DTEType              `json:"documentType"`
	DocumentNumber  *identification.DocumentNumber `json:"documentNumber"`
	Name            *string                        `json:"name"`
	Email           *base.Email                    `json:"email"`
	ReplacementCode *identification.GenerationCode `json:"replacementCode,omitempty"`
	IVAAmount       *financial.Amount              `json:"IVAAmount,omitempty"`
	Phone           *base.Phone                    `json:"phone,omitempty"`
}
