package interfaces

// PaymentTypeGetter is an interface that defines the getter methods that a payment type must implement
type PaymentTypeGetter interface {
	GetCode() string
	GetAmount() float64
	GetReference() string
	GetTerm() *string
	GetPeriod() *int
	GetPeriodPointer() *int
}

// PaymentTypeSetter is an interface that defines the setter methods that a payment type must implement
type PaymentTypeSetter interface {
	SetCode(code string) error
	SetAmount(amount float64) error
	SetReference(reference string) error
	SetTerm(term *string) error
	SetPeriod(period *int) error
}

// PaymentType is an interface that combines the getters and setters of PaymentType
type PaymentType interface {
	PaymentTypeGetter
	PaymentTypeSetter
}
