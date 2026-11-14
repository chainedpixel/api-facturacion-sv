package db_models

// DTEDetails is a structure that represents the details of a DTE stored in the database.
// It is used to store DTE details in the database and retrieve them for processing.
// The details of a DTE are stored in the database for subsequent processing and submission to Hacienda.
//
// For more information about the structure of a DTE see: https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "3. Manual Funcional del Sistema de Transmisión"
// page 53 of the PDF document.
//
// json_data guarda la estructura del DTE (MySQL puede reordenar sus llaves). signed_document guarda el JWS
// transmitido, que es la fuente del Archivo DTE (estructura firmada + firmaElectronica + selloRecibido).
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
	// SignedDocument es el JWS transmitido a Hacienda; su payload es la estructura firmada exacta
	// (json_data no sirve para el Archivo DTE porque MySQL reordena las llaves del tipo JSON)
	SignedDocument   *string `gorm:"column:signed_document;type:longtext"`
	ProcessingDate   *string `gorm:"column:processing_date;type:varchar(30)"`
	HaciendaResponse *string `gorm:"column:hacienda_response;type:text"`

	BalanceControl *DTEBalanceControl `gorm:"foreignKey:OriginalDTEID;references:ID"`
}

func (DTEDetails) TableName() string {
	return "dte_details"
}
