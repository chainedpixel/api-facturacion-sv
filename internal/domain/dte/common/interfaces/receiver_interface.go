package interfaces

// ReceiverGetter is an interface that defines the getter methods that a receiver must implement
type ReceiverGetter interface {
	GetName() *string
	GetDocumentType() *string
	GetDocumentNumber() *string
	GetAddress() Address
	GetEmail() *string
	GetPhone() *string
	GetNRC() *string
	GetNIT() *string
	GetActivityCode() *string
	GetActivityDescription() *string
	GetCommercialName() *string
}

// ReceiverSetter is an interface that defines the setter methods that a receiver must implement
type ReceiverSetter interface {
	SetName(name *string) error
	SetDocumentType(documentType *string) error
	SetDocumentNumber(documentNumber *string) error
	SetAddress(address Address) error
	SetEmail(email *string) error
	SetPhone(phone *string) error
	SetNRC(nrc *string) error
	SetNIT(nit *string) error
	SetActivityCode(activityCode *string) error
	SetActivityDescription(activityDescription *string) error
	SetCommercialName(commercialName *string) error
}

// Receiver is an interface that combines the getters and setters of Receiver
type Receiver interface {
	ReceiverGetter
	ReceiverSetter
}
