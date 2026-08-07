package invalidation_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
)

type InvalidationReason struct {
	Type               document.InvalidationType     `json:"type"`
	ResponsibleName    string                        `json:"responsibleName"`
	ResponsibleDocType document.DTEType              `json:"responsibleDocType"`
	ResponsibleDocNum  identification.DocumentNumber `json:"responsibleDocNum"`
	RequesterName      string                        `json:"requesterName"`
	RequesterDocType   document.DTEType              `json:"requesterDocType"`
	RequesterDocNum    identification.DocumentNumber `json:"requesterDocNum"`
	Reason             *document.InvalidationReason  `json:"reason"`
}
