package structs

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"

type CreateInvalidationRequest struct {
	GenerationCode            string         `json:"generation_code"`
	Reason                    *ReasonRequest `json:"reason"`
	ReplacementGenerationCode *string        `json:"replacement_generation_code,omitempty"`
}

type ReasonRequest struct {
	Type               int     `json:"type"`
	ResponsibleName    string  `json:"responsible_name"`
	ResponsibleDocType string  `json:"responsible_doc_type"`
	ResponsibleNumDoc  string  `json:"responsible_num_doc"`
	RequestorName      string  `json:"requestor_name"`
	RequestorDocType   string  `json:"requestor_doc_type"`
	RequestorNumDoc    string  `json:"requestor_num_doc"`
	Reason             *string `json:"reason_field,omitempty"`
}

type InvalidationExtractor struct {
	Document *models.DTEDocument `json:"*Models.DTEDocument"`
}
