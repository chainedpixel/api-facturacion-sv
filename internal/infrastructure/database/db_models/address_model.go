package db_models

// Address represents the address of the headquarters, branch office, or agencies of DTE issuers in the database.
// Fields such as Municipality and Department are two-letter codes because they refer to the department
// and municipality codes of El Salvador required by Hacienda. For more information about municipality and department codes
//
// of El Salvador see: https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "2. Catálogos- Sistema de Transmisión"
// pages 6-8 of the PDF document and review internal/domain/dte/common/value_objects/location/department.go
// and internal/domain/dte/common/value_objects/location/municipality.go
type Address struct {
	ID           uint   `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	BranchID     uint   `gorm:"column:branch_id;type:uint;not null;index:idx_address_branch"`
	Municipality string `gorm:"column:municipality;type:varchar(2);not null"`
	Department   string `gorm:"column:department;type:varchar(2);not null"`
	District     string `gorm:"column:district;type:varchar(2);not null;default:'01'"`
	Complement   string `gorm:"column:complement;type:varchar(200);not null"`

	Branch *BranchOffice `gorm:"foreignKey:BranchID;references:ID"`
}

func (Address) TableName() string {
	return "addresses"
}
