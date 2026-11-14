package structs

// CreateFSERequest structure for mapping the creation of an FSE
type CreateFSERequest struct {
	Items      []FSEItemRequest    `json:"items"`
	Receiver   *FSEReceiverRequest `json:"excluded_subject"`
	Summary    *FSESummaryRequest  `json:"summary"`
	Appendixes []AppendixRequest   `json:"appendixes,omitempty"`
}

// FSEItemRequest structure for mapping an item of an FSE
type FSEItemRequest struct {
	ItemRequest
	Purchase float64 `json:"purchase"`
}

// FSESummaryRequest structure for mapping the summary of an FSE
type FSESummaryRequest struct {
	SummaryRequest
	TotalPurchase   float64 `json:"total_purchase"`
	IVARetention    float64 `json:"iva_retention"`
	IncomeRetention float64 `json:"income_retention"`
	Observations    *string `json:"observations,omitempty"`
}

// FSEReceiverRequest structure for mapping the excluded subject receiver of an FSE
type FSEReceiverRequest struct {
	ReceiverRequest
	DocumentType        string  `json:"document_type"`
	DocumentNumber      string  `json:"document_number"`
	ActivityCode        *string `json:"activity_code"`
	ActivityDescription *string `json:"activity_description"`
}
