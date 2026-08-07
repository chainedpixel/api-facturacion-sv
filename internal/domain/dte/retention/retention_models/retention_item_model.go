package retention_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
)

type RetentionItem struct {
	Number          item.ItemNumber
	DTEType         document.DTEType
	DocumentType    document.OperationType
	DocumentNumber  document.DocumentNumber
	EmissionDate    temporal.EmissionDate
	RetentionAmount financial.Amount
	ReceptionCodeMH document.RetentionCode
	RetentionIVA    financial.Amount
	Description     string
}
