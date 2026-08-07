package db_models

import "time"

type ReservedSequenceNumber struct {
	ID                    uint      `gorm:"primaryKey;autoIncrement"`
	BranchID              uint      `gorm:"not null;index:idx_reservation_unique,unique;index:idx_branch_id"`
	DTEType               string    `gorm:"type:varchar(2);not null;index:idx_reservation_unique,unique"`
	SequenceNumber        uint      `gorm:"not null;index:idx_reservation_unique,unique"`
	Year                  int       `gorm:"not null;index:idx_reservation_unique,unique"`
	Status                string    `gorm:"type:varchar(20);not null;default:'RESERVED';index:idx_status"`
	DocumentID            *string   `gorm:"type:varchar(36);index:idx_document_id"`
	ReservedAt            time.Time `gorm:"not null;autoCreateTime"`
	ConfirmedAt           *time.Time
	ReleasedAt            *time.Time
	ExpiresAt             *time.Time `gorm:"index:idx_expires"`
	RejectionReason       *string    `gorm:"type:text"`
	HaciendaCode          *string    `gorm:"type:varchar(10)"`
	IsContingency         bool       `gorm:"not null;default:false;index:idx_contingency"`
	ContingencyDocumentID *string    `gorm:"type:varchar(36)"`
}

func (ReservedSequenceNumber) TableName() string {
	return "reserved_sequence_numbers"
}
