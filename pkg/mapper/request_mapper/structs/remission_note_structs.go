package structs

// CreateRemissionNoteRequest represents the request to create a Remission Note
type CreateRemissionNoteRequest struct {
	Receiver       *RemissionNoteReceiverRequest `json:"receiver" validate:"required"`
	Items          []*RemissionNoteItemRequest   `json:"items" validate:"required,min=1,dive"`
	Summary        *RemissionNoteSummaryRequest  `json:"summary" validate:"required"`
	Appendixes     []*AppendixRequest            `json:"appendixes,omitempty"`
	ThirdPartySale *ThirdPartySaleRequest        `json:"third_party_sale,omitempty"`
	RelatedDocs    []*RelatedDocRequest          `json:"related_docs,omitempty"`
}

// RemissionNoteReceiverRequest extends ReceiverRequest with specific bienTitulo field
type RemissionNoteReceiverRequest struct {
	*ReceiverRequest
	BienTitulo *string `json:"real_state" validate:"required,len=2"`
}

// RemissionNoteItemRequest represents an item of a Remission Note
type RemissionNoteItemRequest struct {
	ItemNumber     *int     `json:"number" validate:"required,min=1"`
	ItemType       *int     `json:"type" validate:"required,oneof=1 2 3 4"`
	DocumentNumber *string  `json:"document_number,omitempty"`
	Code           *string  `json:"code,omitempty"`
	TributeCode    *string  `json:"tax_code,omitempty"`
	Description    *string  `json:"description" validate:"required,min=1,max=1000"`
	Quantity       *float64 `json:"quantity" validate:"required,gt=0"`
	UnitMeasure    *int     `json:"unit_measure" validate:"required,min=1,max=99"`
	UnitPrice      *float64 `json:"unit_price" validate:"required,gte=0"`
	DiscountAmount *float64 `json:"discount" validate:"gte=0"`
	NonSubjectSale *float64 `json:"non_subject_sale" validate:"gte=0"`
	ExemptSale     *float64 `json:"exempt_sale" validate:"gte=0"`
	TaxedSale      *float64 `json:"taxed_sale" validate:"gte=0"`
	Tributes       []string `json:"tributes,omitempty"`
}

// RemissionNoteSummaryRequest represents the summary of a Remission Note
type RemissionNoteSummaryRequest struct {
	NonSubjectTotal    *float64          `json:"non_subject_total" validate:"gte=0"`
	ExemptTotal        *float64          `json:"exempt_total" validate:"gte=0"`
	TaxedTotal         *float64          `json:"taxed_total" validate:"gte=0"`
	SubtotalSales      *float64          `json:"subtotal_sales" validate:"required,gte=0"`
	NonSubjectDiscount *float64          `json:"non_subject_discount" validate:"gte=0"`
	ExemptDiscount     *float64          `json:"exempt_discount" validate:"gte=0"`
	TaxedDiscount      *float64          `json:"taxed_discount" validate:"gte=0"`
	DiscountPercent    *float64          `json:"discount_percent,omitempty" validate:"omitempty,gte=0,lte=100"`
	TotalDiscount      *float64          `json:"total_discount" validate:"gte=0"`
	Tributes           []*TaxRequest     `json:"tributes,omitempty"`
	Subtotal           *float64          `json:"subtotal" validate:"required,gte=0"`
	TotalAmount        *float64          `json:"total_amount" validate:"required,gte=0"`
	AmountInWords      *string           `json:"amount_in_words" validate:"required,max=200"`
	Payments           []*PaymentRequest `json:"payments,omitempty"`
	Observations       *string           `json:"observations,omitempty" validate:"omitempty,max=3000"`
}
