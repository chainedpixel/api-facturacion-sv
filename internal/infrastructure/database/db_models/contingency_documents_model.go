package db_models

import "time"

// ContingencyDocument represents a contingency document in the database.
// It is used to store documents that could not be sent to Hacienda and must be sent on a deferred basis.
// The application automatically retries sending these documents.
// The Type field indicates the type of contingency document. For more information about contingency types
//
// see: https://factura.gob.sv/informacion-tecnica-y-funcional/
// in the section "Documentos de Sistema de Transmisión DTE", document: "2. Catálogos- Sistema de Transmisión"
// page 5 of the PDF document and review /internal/domain/dte/common/constants/contingency_document_types.go
type ContingencyDocument struct {
	ID              string    `gorm:"column:id;type:varchar(36);primaryKey;not null"`
	DocumentID      string    `gorm:"column:document_id;type:varchar(36);not null;index"`
	BranchID        uint      `gorm:"column:branch_id;type:uint;not null;index:idx_contingency_branch"`
	ContingencyType int8      `gorm:"column:type;contingency_type:tinyint;not null;index"`
	Reason          string    `gorm:"column:reason;type:varchar(150);not null;index"`
	BatchID         *string   `gorm:"column:batch_id;type:varchar(36);index"`
	MHBatchID       *string   `gorm:"column:mh_batch_id;type:varchar(36)"`
	Observations    *string   `gorm:"column:observations;type:text"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;index:idx_contingency_date"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP"`

	Document *DTEDetails   `gorm:"foreignKey:DocumentID;references:ID"`
	Branch   *BranchOffice `gorm:"foreignKey:BranchID;references:ID"`
}

func (ContingencyDocument) TableName() string {
	return "contingency_documents"
}
