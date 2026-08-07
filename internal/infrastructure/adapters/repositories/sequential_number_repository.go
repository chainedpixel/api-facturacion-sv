package repositories

import (
	"context"
	"errors"

	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"gorm.io/gorm"
)

type ControlNumberRepository struct {
	db *gorm.DB
}

// NewControlNumberRepository creates an instance of ControlNumberRepository. Receives a gorm.DB instance.
func NewControlNumberRepository(db *gorm.DB) ports.SequentialNumberRepositoryPort {
	return &ControlNumberRepository{db: db}
}

// GetNext retrieves the next control number for a DTE type, system NIT and establishment code.
func (r *ControlNumberRepository) GetNext(ctx context.Context, dteType string, branchID uint) (int, error) {
	currentYear := utils.TimeNow().Year()
	var sequence db_models.ControlNumberSequence

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("branch_id = ? AND dte_type = ? AND year = ?", branchID, dteType, currentYear).
			First(&sequence)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			sequence = db_models.ControlNumberSequence{
				BranchID:   branchID,
				DTEType:    dteType,
				LastNumber: 0,
				Year:       currentYear,
			}
		}

		sequence.LastNumber++

		if result.Error == nil {
			if err := tx.Save(&sequence).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(&sequence).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		logs.Error("Failed to get next control number", map[string]interface{}{
			"dteType":  dteType,
			"branchID": branchID,
			"error":    err.Error(),
		})
		return 0, err
	}

	return sequence.LastNumber, nil
}
