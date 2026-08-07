package db_models

// DTEDetails is a structure that represents the details of a DTE stored in the database.
// It is used to store DTE details in the database and retrieve them for processing.
// The details of a DTE are stored in the database for subsequent processing and submission to Hacienda.
//
// For more information about the structure of a DTE see: https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "3. Manual Funcional del Sistema de Transmisión"
// page 53 of the PDF document.
//
// Note: The DTE is not stored signed; it is only stored in JSON format, the format prior to signing. The signature is the DTE itself
// signed with the issuer's private key in JWT format.
//
// The DTEType field indicates the type of DTE. For more information about DTE types see:
// https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "2. Catálogos- Sistema de Transmisión"
// page 5 of the PDF document and review /internal/domain/dte/common/constants/dte_type.go
type DTEDetails struct {
	ID             string  `gorm:"column:id;varchar(36);primaryKey;not null;index:idx_dte_details"`
	DTEType        string  `gorm:"column:dte_type;varchar(2);not null;index:idx_dte_type"`
	ControlNumber  string  `gorm:"column:control_number;varchar(30);not null;index"`
	ReceptionStamp *string `gorm:"column:reception_stamp;varchar(40)"`
	Transmission   string  `gorm:"column:transmission;varchar(15);not null"`
	Status         string  `gorm:"column:status;varchar(15);not null;index"`
	JSONData       string  `gorm:"column:json_data;type:json;not null"`

	BalanceControl *DTEBalanceControl `gorm:"foreignKey:OriginalDTEID;references:ID"`
}

func (DTEDetails) TableName() string {
	return "dte_details"
}
