package interfaces

import "time"

// Identification is an interface that defines the methods that must be implemented by an Identification object
type Identification interface {
	IdentificationGetter
	IdentificationSetter
}

type IdentificationGetter interface {
	GetVersion() int
	GetAmbient() string
	GetDTEType() string
	GetControlNumber() string
	GetGenerationCode() string
	GetModelType() int
	GetOperationType() int
	GetEmissionDate() time.Time
	GetEmissionTime() time.Time
	GetCurrency() string
	GetContingencyType() *int
	GetContingencyReason() *string
}

type IdentificationSetter interface {
	SetControlNumber(controlNumber string) error
	GenerateCode() error
	SetVersion(version int) error
	SetAmbient(ambient string) error
	SetDTEType(dteType string) error
	SetDTETypeForce(dteType string)
	SetModelType(modelType int) error
	SetOperationType(operationType int) error
	SetEmissionDate(emissionDate time.Time) error
	SetEmissionTime(emissionTime time.Time) error
	SetCurrency(currency string) error
	SetContingencyType(contingencyType *int) error
	SetContingencyReason(contingencyReason *string) error
}
