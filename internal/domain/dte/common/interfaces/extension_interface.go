package interfaces

// ExtensionGetter is an interface that defines the getter methods that must be implemented by an extension
type ExtensionGetter interface {
	GetDeliveryName() string
	GetDeliveryDocument() string
	GetReceiverName() string
	GetReceiverDocument() string
	GetObservation() *string
	GetVehiculePlate() *string
}

// ExtensionSetter is an interface that defines the setter methods that must be implemented by an extension
type ExtensionSetter interface {
	SetDeliveryName(deliveryName string) error
	SetDeliveryDocument(deliveryDocument string) error
	SetReceiverName(receiverName string) error
	SetReceiverDocument(receiverDocument string) error
	SetObservation(observation *string) error
	SetVehiculePlate(vehiculePlate *string) error
}

// Extension is an interface that combines the getters and setters of Extension
type Extension interface {
	ExtensionGetter
	ExtensionSetter
}
