package structs

type RetentionItem struct {
	DocumentType   int      `json:"type"`
	DocumentNumber string   `json:"document_number"`
	Description    string   `json:"description"`
	RetentionCode  string   `json:"retention_code"`
	IvaAmount      *float64 `json:"iva_amount,omitempty"`
	TaxedAmount    *float64 `json:"taxed_amount,omitempty"`
	EmissionDate   *string  `json:"emission_date,omitempty"`
	DTEType        *string  `json:"dte_type,omitempty"`
}

type RetentionSummary struct {
	TotalRetentionAmount float64 `json:"total_retention_amount"`
	TotalRetentionIVA    float64 `json:"total_retention_iva"`
}

type CreateRetentionRequest struct {
	Items      []RetentionItem   `json:"items"`
	Summary    *RetentionSummary `json:"summary,omitempty"`
	Receiver   *ReceiverRequest  `json:"receiver,omitempty"`
	Extension  *ExtensionRequest `json:"extension,omitempty"`
	Appendixes []AppendixRequest `json:"appendixes,omitempty"`
}
