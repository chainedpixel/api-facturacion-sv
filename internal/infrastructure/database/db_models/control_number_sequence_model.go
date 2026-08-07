package db_models

import "time"

// ControlNumberSequence represents the control sequence for the numbers of issued electronic documents.
// This sequence is used to generate the control number of electronic documents by document type and year.
// They represent the last 15 digits of the control number of an electronic document.
// This sequence resets each year and per document type. It increments by 1 for each electronic document issued.
//
// For more information see: https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "3. Manual Funcional del Sistema de Transmisión"
// page 16 of the PDF document.
//
// The DTEType field represents the type of electronic document, two characters long, as required by the Ministry of Finance.
// For more information see: https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "2. Catálogos- Sistema de Transmisión"
// page 5 of the PDF document and review /internal/domain/dte/common/constants/dte_type.go
type ControlNumberSequence struct {
	ID         uint      `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	BranchID   uint      `gorm:"column:branch_id;type:uint;not null;uniqueIndex:idx_branch_dte_year,priority:1"`
	DTEType    string    `gorm:"column:dte_type;type:varchar(2);not null;uniqueIndex:idx_branch_dte_year,priority:2"`
	Year       int       `gorm:"column:year;type:int;not null;uniqueIndex:idx_branch_dte_year,priority:3;index:idx_sequence_year"`
	LastNumber int       `gorm:"column:last_number;type:int;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP"`

	Branch *BranchOffice `gorm:"foreignKey:BranchID;references:ID"`
}

func (ControlNumberSequence) TableName() string {
	return "control_number_sequences"
}
