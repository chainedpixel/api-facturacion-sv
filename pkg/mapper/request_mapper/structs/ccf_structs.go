package structs

type CreateCreditFiscalRequest struct {
	Items          []CreditItemRequest    `json:"items"`
	Receiver       *ReceiverRequest       `json:"receiver"`
	ModelType      int                    `json:"model_type"`
	Summary        *CreditSummaryRequest  `json:"summary"`
	ThirdPartySale *ThirdPartySaleRequest `json:"third_party_sale,omitempty"`
	Extension      *ExtensionRequest      `json:"extension,omitempty"`
	Payments       []PaymentRequest       `json:"payments,omitempty"`
	OtherDocs      []OtherDocRequest      `json:"other_docs,omitempty"`
	RelatedDocs    []RelatedDocRequest    `json:"related_docs,omitempty"`
	Appendixes     []AppendixRequest      `json:"appendixes,omitempty"`
}

// CreditItemRequest structure for mapping an item of Comprobante de Crédito Fiscal
type CreditItemRequest struct {
	ItemRequest
	NonSubjectSale float64 `json:"non_subject_sale"`
	ExemptSale     float64 `json:"exempt_sale"`
	TaxedSale      float64 `json:"taxed_sale"`
	SuggestedPrice float64 `json:"suggested_price"`
	NonTaxed       float64 `json:"non_taxed"`
}

// CreditSummaryRequest structure for mapping the summary of a Comprobante de Crédito Fiscal
type CreditSummaryRequest struct {
	SummaryRequest
	TaxedDiscount   float64 `json:"taxed_discount"`
	IVAPerception   float64 `json:"iva_perception"`
	IVARetention    float64 `json:"iva_retention"`
	IncomeRetention float64 `json:"income_retention"`
	TotalIVA        float64 `json:"total_iva"`
	BalanceInFavor  float64 `json:"balance_in_favor"`
}
