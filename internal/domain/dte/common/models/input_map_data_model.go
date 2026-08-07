package models

type InputDataCommon struct {
	Identification *Identification
	Issuer         *Issuer
	Receiver       *Receiver
	Extension      *Extension        `json:"extension,omitempty"`
	RelatedDocs    []RelatedDocument `json:"relatedDocs,omitempty"`
	OtherDocs      []OtherDocument   `json:"otherDocs,omitempty"`
	ThirdPartySale *ThirdPartySale   `json:"thirdPartySale,omitempty"`
	Appendixes     []Appendix        `json:"appendixes,omitempty"`
}
