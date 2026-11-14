package structs

type CreateCreditNoteRequest struct {
	Items          []CreditNoteItemRequest   `json:"items"`
	Receiver       *ReceiverRequest          `json:"receiver"`
	ModelType      int                       `json:"model_type"`
	Summary        *CreditNoteSummaryRequest `json:"summary"`
	ThirdPartySale *ThirdPartySaleRequest    `json:"third_party_sale,omitempty"`
	Payments       []PaymentRequest          `json:"payments,omitempty"`
	OtherDocs      []OtherDocRequest         `json:"other_docs,omitempty"`
	RelatedDocs    []RelatedDocRequest       `json:"related_docs,omitempty"`
	Appendixes     []AppendixRequest         `json:"appendixes,omitempty"`
}

// CreditNoteItemRequest structure for mapping an item of a Credit Note
type CreditNoteItemRequest struct {
	ItemRequest
	NonSubjectSale float64 `json:"non_subject_sale"`
	ExemptSale     float64 `json:"exempt_sale"`
	TaxedSale      float64 `json:"taxed_sale"`
}

// CreditNoteSummaryRequest structure for mapping the summary of a Credit Note
type CreditNoteSummaryRequest struct {
	SummaryRequest
	TaxedDiscount   float64 `json:"taxed_discount"`
	IVAPerception   float64 `json:"iva_perception"`
	IVARetention    float64 `json:"iva_retention"`
	IncomeRetention float64 `json:"income_retention"`
}
