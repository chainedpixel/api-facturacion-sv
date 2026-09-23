package structs

// AddressRequest structure for mapping the address of a receiver
type AddressRequest struct {
	Department   string `json:"department"`
	Municipality string `json:"municipality"`
	District     string `json:"district"`
	Complement   string `json:"complement"`
}

// ExtensionRequest structure for mapping the extension of a document
type ExtensionRequest struct {
	DeliveryName     string  `json:"delivery_name"`
	DeliveryDocument string  `json:"delivery_document"`
	ReceiverName     string  `json:"receiver_name"`
	ReceiverDocument string  `json:"receiver_document"`
	Observation      *string `json:"observation,omitempty"`
	VehiculePlate    *string `json:"vehicule_plate,omitempty"`
}

// ReceiverRequest structure for mapping the receiver of a document
type ReceiverRequest struct {
	DocumentType   *string         `json:"document_type,omitempty"`
	DocumentNumber *string         `json:"document_number,omitempty"`
	Name           *string         `json:"name,omitempty"`
	NRC            *string         `json:"nrc,omitempty"`
	NIT            *string         `json:"nit,omitempty"`
	Address        *AddressRequest `json:"address,omitempty"`
	Phone          *string         `json:"phone,omitempty"`
	Email          *string         `json:"email,omitempty"`
	ActivityCode   *string         `json:"activity_code,omitempty"`
	ActivityDesc   *string         `json:"activity_description,omitempty"`
	CommercialName *string         `json:"commercial_name,omitempty"`
}

// ItemRequest structure for mapping an item of a document
type ItemRequest struct {
	Number      int      `json:"number"`
	Type        int      `json:"type"`
	Description string   `json:"description"`
	Quantity    float64  `json:"quantity"`
	UnitMeasure int      `json:"unit_measure"`
	UnitPrice   float64  `json:"unit_price"`
	Discount    float64  `json:"discount"`
	Code        *string  `json:"code,omitempty"`
	TaxCode     *string  `json:"tax_code,omitempty"`
	RelatedDoc  *string  `json:"related_doc,omitempty"`
	Taxes       []string `json:"taxes,omitempty"`
}

// SummaryRequest structure for mapping the summary of a document
type SummaryRequest struct {
	TotalNonSubject    float64          `json:"total_non_subject"`
	TotalExempt        float64          `json:"total_exempt"`
	TotalTaxed         float64          `json:"total_taxed"`
	SubTotal           float64          `json:"sub_total"`
	NonSubjectDiscount float64          `json:"non_subject_discount"`
	ExemptDiscount     float64          `json:"exempt_discount"`
	DiscountPercentage float64          `json:"discount_percentage"`
	TotalDiscount      float64          `json:"total_discount"`
	TotalOperation     float64          `json:"total_operation"`
	TotalNonTaxed      float64          `json:"total_non_taxed"`
	SubTotalSales      float64          `json:"sub_total_sales"`
	TotalToPay         float64          `json:"total_to_pay"`
	OperationCondition int              `json:"operation_condition"`
	Taxes              []TaxRequest     `json:"taxes,omitempty"`
	PaymentTypes       []PaymentRequest `json:"payment_types"`
	TotalInWords       *string          `json:"total_in_words,omitempty"`
}

// TaxRequest structure for mapping a tax of a document
type TaxRequest struct {
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
}

// AppendixRequest structure for mapping an appendix of a document
type AppendixRequest struct {
	Field string `json:"field"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// PaymentRequest structure for mapping a payment of a document
type PaymentRequest struct {
	Code      string  `json:"code"`
	Amount    float64 `json:"amount"`
	Period    *int    `json:"period,omitempty"`
	Term      *string `json:"term,omitempty"`
	Reference *string `json:"reference,omitempty"`
}

// RelatedDocRequest structure for mapping a related document of an invoice
type RelatedDocRequest struct {
	DocumentType   string `json:"document_type"`
	GenerationType int    `json:"generation_type"`
	DocumentNumber string `json:"document_number"`
	EmissionDate   string `json:"emission_date"`
}

// OtherDocRequest structure for mapping an other document of an invoice
type OtherDocRequest struct {
	DocumentCode int            `json:"document_code"`
	Description  *string        `json:"description,omitempty"`
	Detail       *string        `json:"detail,omitempty"`
	Doctor       *DoctorRequest `json:"doctor,omitempty"`
}

// DoctorRequest structure for mapping a doctor of an invoice
type DoctorRequest struct {
	Name              string  `json:"name"`
	NIT               *string `json:"nit,omitempty"`
	IdentificationDoc *string `json:"identification,omitempty"`
	ServiceType       int     `json:"service_type"`
}

// ThirdPartySaleRequest structure for mapping a third-party sale of an invoice
type ThirdPartySaleRequest struct {
	NIT  string `json:"nit"`
	Name string `json:"name"`
}
