package db_models

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"gorm.io/gorm"
)

// DTEBalanceTransaction represents a balance transaction for a DTE (Documento Tributario Electrónico).
// Its purpose is to store DTE balance transactions in the database and retrieve them for processing.
// This table records Credit and Debit Note transactions that affect the balance of a DTE.
type DTEBalanceTransaction struct {
	ID                   uint      `gorm:"primaryKey;autoIncrement:true;not null;index:idx_dte_balance_transaction"`
	BalanceControlID     uint      `gorm:"column:balance_control_id;type:int;not null;index:idx_dte_balance_control"`
	AdjustmentDocumentID string    `gorm:"column:adjustment_document_id;type:varchar(36);not null;index:idx_dte_adjustment_document"`
	TransactionType      string    `gorm:"column:transaction_type;type:varchar(20);not null;index:idx_dte_transaction_type"`
	TaxedAmount          float64   `gorm:"column:taxed_amount;type:decimal(18,2);not null"`
	ExemptAmount         float64   `gorm:"column:exempt_amount;type:decimal(18,2);not null"`
	NotSubjectAmount     float64   `gorm:"column:not_subject_amount;type:decimal(18,2);not null"`
	CreatedAt            time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP"`

	BalanceControl     *DTEBalanceControl `gorm:"foreignKey:BalanceControlID;references:ID"`
	AdjustmentDocument *DTEDetails        `gorm:"foreignKey:AdjustmentDocumentID;references:ID"`
}

func (DTEBalanceTransaction) TableName() string {
	return "dte_balance_transactions"
}

func (d *DTEBalanceTransaction) AfterCreate(tx *gorm.DB) error {
	if d.TransactionType == constants.NotaCreditoElectronica {
		d.BalanceControl.RemainingTaxedAmount -= d.TaxedAmount
		d.BalanceControl.RemainingExemptAmount -= d.ExemptAmount
		d.BalanceControl.RemainingNotSubjectAmount -= d.NotSubjectAmount
	} else {
		d.BalanceControl.RemainingTaxedAmount += d.TaxedAmount
		d.BalanceControl.RemainingExemptAmount += d.ExemptAmount
		d.BalanceControl.RemainingNotSubjectAmount += d.NotSubjectAmount
	}

	err := tx.Save(d.BalanceControl).Error
	if err != nil {
		logs.Error("Error updating DTE balance control", map[string]interface{}{
			"error":       err.Error(),
			"document_id": d.AdjustmentDocumentID,
			"branch_id":   d.BalanceControl.BranchID,
			"dteType":     d.AdjustmentDocument.DTEType,
		})
		return err
	}

	return nil
}
