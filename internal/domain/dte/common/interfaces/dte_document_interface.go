package interfaces

// DTEDocumentGetter is an interface that defines the getter methods that must be implemented by a DTE document
type DTEDocumentGetter interface {
	GetIdentification() Identification
	GetAppendix() []Appendix
	GetExtension() Extension
	GetIssuer() Issuer
	GetReceiver() Receiver
	GetItems() []Item
	GetSummary() Summary
	GetRelatedDocuments() []RelatedDocument
	GetOtherDocuments() []OtherDocuments
	GetThirdPartySale() ThirdPartySale
}

// DTEDocumentSetter is an interface that defines the setter methods that must be implemented by a DTE document
type DTEDocumentSetter interface {
	SetIdentification(identification Identification) error
	SetAppendix(appendix []Appendix) error
	SetExtension(extension Extension) error
	SetIssuer(issuer Issuer) error
	SetReceiver(receiver Receiver) error
	SetItems(items []Item) error
	SetSummary(summary Summary) error
	SetRelatedDocuments(relatedDocuments []RelatedDocument) error
	SetOtherDocuments(otherDocuments []OtherDocuments) error
	SetThirdPartySale(thirdPartySale ThirdPartySale) error
}

// DTEDocument is an interface that combines the getters and setters of DTEDocument
type DTEDocument interface {
	DTEDocumentGetter
	DTEDocumentSetter
	Validate() error
}
