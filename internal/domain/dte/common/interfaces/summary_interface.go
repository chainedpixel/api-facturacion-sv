package interfaces

// Summary is an interface that defines the methods that a summary must implement
type Summary interface {
	SummaryGetters
	SummarySetters
}

type SummaryGetters interface {
	GetTotalNonSubject() float64
	GetTotalExempt() float64
	GetTotalTaxed() float64
	GetSubTotal() float64
	GetSubtotalSales() float64
	GetNonSubjectDiscount() float64
	GetExemptDiscount() float64
	GetDiscountPercentage() float64
	GetTotalDiscount() float64
	GetTotalTaxes() []Tax
	GetTotalOperation() float64
	GetTotalNotTaxed() float64
	GetPaymentTypes() []PaymentType
	GetOperationCondition() int
	GetElectronicPayment() *string
	GetTotalInWords() string
	GetTotalToPay() float64
	GetObservations() *string
}

// SummarySetters is an interface that defines the setter methods that a summary must implement
type SummarySetters interface {
	SetTotalNonSubject(totalNonSubject float64) error
	SetTotalExempt(totalExempt float64) error
	SetTotalTaxed(totalTaxed float64) error
	SetSubTotal(subTotal float64) error
	SetSubtotalSales(subtotalSales float64) error
	SetNonSubjectDiscount(nonSubjectDiscount float64) error
	SetExemptDiscount(exemptDiscount float64) error
	SetDiscountPercentage(discountPercentage float64) error
	SetTotalDiscount(totalDiscount float64) error
	SetTotalTaxes(totalTaxes []Tax) error
	SetTotalOperation(totalOperation float64) error
	SetTotalNotTaxed(totalNotTaxed float64) error
	SetPaymentTypes(paymentTypes []PaymentType) error
	SetOperationCondition(operationCondition int) error
	SetElectronicPayment(electronicPayment *string) error
	SetTotalInWords(totalInWords string) error
	SetTotalToPay(totalToPay float64) error
	SetForceTotalToPay(totalToPay float64)
	SetObservations(observations *string) error
}

// SummaryManager is an interface that combines the getters and setters of Summary
type SummaryManager interface {
	SummaryGetters
	SummarySetters
}
