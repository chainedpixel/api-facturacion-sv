package db_models

import "time"

// DTEBalanceControl represents the balance control for a DTE (Documento Tributario Electrónico).
// It is used to store the balance of a DTE in the database and retrieve it for processing in a more efficient manner,
// which also allows for more effective tracking of DTE balances when adjustments such as credit or debit notes are made.
type DTEBalanceControl struct {
	ID                            uint      `gorm:"primaryKey;autoIncrement:true;not null;index:idx_dte_balance_control"`
	BranchID                      uint      `gorm:"column:branch_id;type:uint;not null;index:idx_dte_branch"`
	OriginalDTEID                 string    `gorm:"column:original_dte_id;type:varchar(36);not null;index:idx_dte_original"`
	OriginalTaxedAmount           float64   `gorm:"column:original_taxed_amount;type:decimal(18,2);not null;"`
	OriginalExemptAmount          float64   `gorm:"column:original_exempt_amount;type:decimal(18,2);not null"`
	OriginalTotalNotSubjectAmount float64   `gorm:"column:original_not_subject_amount;type:decimal(18,2);not null"`
	RemainingTaxedAmount          float64   `gorm:"column:remaining_taxed_amount;type:decimal(18,2);not null"`
	RemainingExemptAmount         float64   `gorm:"column:remaining_exempt_amount;type:decimal(18,2);not null"`
	RemainingNotSubjectAmount     float64   `gorm:"column:remaining_not_subject_amount;type:decimal(18,2);not null"`
	CreatedAt                     time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt                     time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP"`

	OriginalDTE  *DTEDetails             `gorm:"foreignKey:OriginalDTEID;references:ID"`
	Branch       *BranchOffice           `gorm:"foreignKey:BranchID;references:ID"`
	Transactions []DTEBalanceTransaction `gorm:"foreignKey:BalanceControlID;references:ID"`
}

func (DTEBalanceControl) TableName() string {
	return "dte_balance_control"
}
