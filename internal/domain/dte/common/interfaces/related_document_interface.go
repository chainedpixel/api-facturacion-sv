package interfaces

import "time"

// RelatedDocumentGetter is an interface that defines the getter methods that must be implemented by related documents
type RelatedDocumentGetter interface {
	GetDocumentType() string
	GetGenerationType() int
	GetDocumentNumber() string
	GetEmissionDate() time.Time
}

// RelatedDocumentSetter is an interface that defines the setter methods that must be implemented by related documents
type RelatedDocumentSetter interface {
	SetDocumentType(documentType string) error
	SetGenerationType(generationType int) error
	SetDocumentNumber(documentNumber string) error
	SetEmissionDate(emissionDate time.Time) error
}

// RelatedDocument is an interface that combines the getters and setters of RelatedDocument
type RelatedDocument interface {
	RelatedDocumentGetter
	RelatedDocumentSetter
}
