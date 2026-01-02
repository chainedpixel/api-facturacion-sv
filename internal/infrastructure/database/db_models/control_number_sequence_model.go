package db_models

import "time"

// ControlNumberSequence representa la secuencia de control de los números de los documentos electrónicos emitidos.
// Esta secuencia se utiliza para generar el número de control de los documentos electrónicos por tipo de documento y año.
// Representan los últimos 15 dígitos del número de control de un documento electrónico.
// Esta secuencia se reinicia cada año y por tipo de documento. Se incrementa en 1 por cada documento electrónico emitido.
//
// Para más información ver: https://factura.gob.sv/informacion-tecnica-y-funcional/
// en la sección de "Documentos de Sistema de Transmisión DTE", documento: "3. Manual Funcional del Sistema de Transmisión"
// página 16 del documento PDF.
//
// El campo DTEType representa el tipo de documento electrónico, de dos caracteres, como exige el Ministerio de Hacienda.
// Para más información ver: https://factura.gob.sv/informacion-tecnica-y-funcional/
// en la sección de "Documentos de Sistema de Transmisión DTE", documento: "2. Catálogos- Sistema de Transmisión"
// página 5 del documento PDF y revisar /internal/domain/dte/common/constants/dte_type.go
type ControlNumberSequence struct {
	ID         uint      `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	BranchID   uint      `gorm:"column:branch_id;type:uint;not null;uniqueIndex:idx_branch_dte_year,priority:1"`
	DTEType    string    `gorm:"column:dte_type;type:varchar(2);not null;uniqueIndex:idx_branch_dte_year,priority:2"`
	Year       int       `gorm:"column:year;type:int;not null;uniqueIndex:idx_branch_dte_year,priority:3;index:idx_sequence_year"`
	LastNumber int       `gorm:"column:last_number;type:int;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP"`

	// Relaciones
	Branch *BranchOffice `gorm:"foreignKey:BranchID;references:ID"`
}

func (ControlNumberSequence) TableName() string {
	return "control_number_sequences"
}
