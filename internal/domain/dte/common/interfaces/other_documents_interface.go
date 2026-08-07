package interfaces

// OtherDocumentsGetter is an interface that defines the getter methods that a document must implement
type OtherDocumentsGetter interface {
	GetAssociatedDocument() int
	GetDescription() string
	GetDetail() string
	GetDoctor() DoctorInfo
}

// OtherDocumentsSetter is an interface that defines the setter methods that a document must implement
type OtherDocumentsSetter interface {
	SetAssociatedDocument(associatedDocument int) error
	SetDescription(description string) error
	SetDetail(detail string) error
	SetDoctor(doctor DoctorInfo) error
}

// OtherDocuments is an interface that combines the getters and setters of OtherDocuments
type OtherDocuments interface {
	OtherDocumentsGetter
	OtherDocumentsSetter
}

// DoctorInfoGetter is an interface that defines the getter methods that a doctor must implement
type DoctorInfoGetter interface {
	GetName() string
	GetServiceType() int
	GetNIT() string
	GetIdentification() string
}

// DoctorInfoSetter is an interface that defines the setter methods that a doctor must implement
type DoctorInfoSetter interface {
	SetName(name string) error
	SetServiceType(serviceType int) error
	SetNIT(nit string) error
	SetIdentification(identification string) error
}

// DoctorInfo is an interface that combines the getters and setters of DoctorInfo
type DoctorInfo interface {
	DoctorInfoGetter
	DoctorInfoSetter
}
