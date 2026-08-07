package db_models

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"gorm.io/gorm"
)

// DTEDocument represents the relationship between an electronic tax document and a branch office.
// It is used to store the relationship between an electronic tax document and a branch office.
// The relationship between an electronic tax document and a branch office is stored in the database
// for subsequent processing and submission to Hacienda.
type DTEDocument struct {
	DocumentID string    `gorm:"column:document_id;type:varchar(36);primaryKey;not null;index:idx_dte_document"`
	BranchID   uint      `gorm:"column:branch_id;type:uint;not null;index:idx_dte_branch"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP;index:idx_dte_date"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP"`

	Branch   *BranchOffice `gorm:"foreignKey:BranchID;references:ID"`
	Document *DTEDetails   `gorm:"foreignKey:DocumentID;references:ID"`
}

func (d *DTEDocument) AfterCreate(tx *gorm.DB) error {
	if constants.ValidAdjustmentDTETypes[d.Document.DTEType] {

		extractor, err := utils.ExtractSummaryTotalAmountsFromStringJSON(d.Document.JSONData)
		if err != nil {
			return err
		}

		dteBalanceControl := &DTEBalanceControl{
			OriginalDTEID:                 d.DocumentID,
			BranchID:                      d.BranchID,
			OriginalTaxedAmount:           extractor.Summary.TotalTaxed,
			OriginalExemptAmount:          extractor.Summary.TotalExempt,
			OriginalTotalNotSubjectAmount: extractor.Summary.TotalNotSubject,
			RemainingTaxedAmount:          extractor.Summary.TotalTaxed,
			RemainingExemptAmount:         extractor.Summary.TotalExempt,
			RemainingNotSubjectAmount:     extractor.Summary.TotalNotSubject,
		}
		if err = tx.Create(dteBalanceControl).Error; err != nil {
			logs.Error("Error creating DTE balance control", map[string]interface{}{
				"error":       err.Error(),
				"document_id": d.DocumentID,
				"branch_id":   d.BranchID,
				"dteType":     d.Document.DTEType,
			})
			return err
		}
	}

	return nil
}

func (DTEDocument) TableName() string {
	return "dte_documents"
}
