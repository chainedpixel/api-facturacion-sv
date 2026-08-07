package interfaces

// ItemGetter is an interface that defines the getter methods that an item must implement
type ItemGetter interface {
	GetQuantity() float64
	GetItemCode() string
	GetDescription() string
	GetType() int
	GetUnitPrice() float64
	GetDiscount() float64
	GetTaxes() []string
	GetRelatedDoc() *string
	GetNumber() int
	GetUnitMeasure() int
}

// ItemSetter is an interface that defines the setter methods that an item must implement
type ItemSetter interface {
	SetQuantity(quantity float64) error
	SetItemCode(itemCode string) error
	SetDescription(description string) error
	SetType(itemType int) error
	SetUnitPrice(unitPrice float64) error
	SetForceUnitPrice(unitPrice float64)
	SetDiscount(discount float64) error
	SetTaxes(taxes []string) error
	SetRelatedDoc(relatedDoc *string) error
	SetForceRelatedDoc(relatedDoc *string)
	SetNumber(number int) error
	SetUnitMeasure(unitMeasure int) error
}

// Item is an interface that combines the getters and setters of Item
type Item interface {
	ItemGetter
	ItemSetter
}
