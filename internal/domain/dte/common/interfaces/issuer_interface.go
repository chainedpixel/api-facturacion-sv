package interfaces

// IssuerGetter is an interface that defines the getter methods that an issuer must implement
type IssuerGetter interface {
	GetName() string
	GetActivityDescription() string
	GetCommercialName() string
	GetNIT() string
	GetNRC() string
	GetActivityCode() string
	GetEstablishmentType() string
	GetAddress() Address
	GetPhone() string
	GetEmail() string
	GetEstablishmentCode() *string
	GetEstablishmentMHCode() *string
	GetPOSCode() *string
	GetPOSMHCode() *string
}

// IssuerSetter is an interface that defines the setter methods that an issuer must implement
type IssuerSetter interface {
	SetName(name string) error
	SetActivityDescription(description string) error
	SetCommercialName(commercialName string) error
	SetNIT(nit string) error
	SetNRC(nrc string) error
	SetActivityCode(activityCode string) error
	SetEstablishmentType(establishmentType string) error
	SetAddress(address Address) error
	SetPhone(phone string) error
	SetEmail(email string) error
	SetEstablishmentCode(establishmentCode *string) error
	SetEstablishmentMHCode(establishmentMHCode *string) error
	SetPOSCode(posCode *string) error
	SetPOSMHCode(posMHCode *string) error
}

// Issuer is an interface that combines the getters and setters of Issuer
type Issuer interface {
	IssuerGetter
	IssuerSetter
}
