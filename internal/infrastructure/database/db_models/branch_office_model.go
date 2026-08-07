package db_models

// BranchOffice represents the structure of the branch_offices table in the database.
// The EstablishmentType field is a 2-character field representing the establishment type required by Hacienda.
// The field indicates whether the location is a branch, headquarters, etc.
//
// For more information about establishment types see:
// https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "2. Catálogos- Sistema de Transmisión"
// page 6 of the PDF document and review /internal/domain/dte/common/constants/establishment_type.go
//
// Fields ending in MH refer to codes provided by Hacienda. If you do not have those Hacienda codes,
// leave the fields blank to avoid legal issues.
type BranchOffice struct {
	ID                  uint    `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	UserID              uint    `gorm:"column:user_id;type:uint;not null;index:idx_branch_offices_user"`
	EstablishmentCode   *string `gorm:"column:establishment_code;type:varchar(10)"`
	EstablishmentCodeMH *string `gorm:"column:establishment_code_mh;type:varchar(4)"`
	Email               *string `gorm:"column:email;type:varchar(255)"`
	APIKey              string  `gorm:"column:api_key;type:varchar(255);not null;uniqueIndex"`
	APISecret           string  `gorm:"column:api_secret;type:varchar(255);not null"`
	Phone               *string `gorm:"column:phone;type:varchar(30)"`
	EstablishmentType   string  `gorm:"column:establishment_type;type:varchar(2);not null;index:idx_branch_est_type"`
	POSCode             *string `gorm:"column:pos_code;type:varchar(15)"`
	POSCodeMH           *string `gorm:"column:pos_code_mh;type:varchar(4)"`
	IsActive            bool    `gorm:"column:is_active;type:tinyint(1);not null;index:idx_branch_offices_active"`

	User    *User    `gorm:"foreignKey:UserID;references:ID"`
	Address *Address `gorm:"foreignKey:BranchID;references:ID"`
}

func (BranchOffice) TableName() string {
	return "branch_offices"
}
