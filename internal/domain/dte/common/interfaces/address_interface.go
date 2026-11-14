package interfaces

// AddressGetter is an interface that defines the getter methods that an address must implement
type AddressGetter interface {
	GetDepartment() string
	GetMunicipality() string
	GetDistrict() string
	GetComplement() string
}

// AddressSetter is an interface that defines the setter methods that an address must implement
type AddressSetter interface {
	SetDepartment(department string) error
	SetMunicipality(municipality string) error
	SetDistrict(district string) error
	SetComplement(complement string) error
}

// Address is an interface that combines the getters and setters of Address
type Address interface {
	AddressGetter
	AddressSetter
}
