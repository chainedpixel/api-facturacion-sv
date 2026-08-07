package interfaces

// ThirdPartySaleGetter is an interface that defines the getter methods that must be implemented by a ThirdPartySale object
type ThirdPartySaleGetter interface {
	GetNIT() string
	GetName() string
}

// ThirdPartySaleSetter is an interface that defines the setter methods that must be implemented by a ThirdPartySale object
type ThirdPartySaleSetter interface {
	SetNIT(nit string) error
	SetName(name string) error
}

// ThirdPartySale is an interface that combines the getters and setters of ThirdPartySale
type ThirdPartySale interface {
	ThirdPartySaleGetter
	ThirdPartySaleSetter
}
