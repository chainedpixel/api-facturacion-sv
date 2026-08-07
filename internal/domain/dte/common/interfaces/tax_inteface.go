package interfaces

// TaxGetter is an interface that defines the getter methods that a tax must implement
type TaxGetter interface {
	GetTotalAmount() float64
	GetCode() string
	GetDescription() string
	GetValue() float64
}

// TaxSetter is an interface that defines the setter methods that a tax must implement
type TaxSetter interface {
	SetTotalAmount(totalAmount float64) error
	SetCode(code string) error
	SetDescription(description string) error
	SetValue(value float64) error
}

// Tax is an interface that combines the getters and setters of Tax
type Tax interface {
	TaxGetter
	TaxSetter
}
