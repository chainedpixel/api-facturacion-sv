package repositories

import (
	"context"
	"errors"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/ports"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/logs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ControlNumberRepository struct {
	db *gorm.DB
}

// NewControlNumberRepository creates an instance of ControlNumberRepository. Receives a gorm.DB instance.
func NewControlNumberRepository(db *gorm.DB) ports.SequentialNumberRepositoryPort {
	return &ControlNumberRepository{db: db}
}

// GetNext retrieves the next control number for a DTE type and branch. The counter row is read with a
// pessimistic lock inside the same transaction that increments it. When the row does not exist yet and a
// concurrent request inserts it first, the locked read is retried once.
func (r *ControlNumberRepository) GetNext(ctx context.Context, dteType string, branchID uint) (int, error) {
	currentYear := utils.TimeNow().Year()

	next, err := r.nextInTransaction(ctx, dteType, branchID, currentYear)
	if err != nil {
		next, err = r.nextInTransaction(ctx, dteType, branchID, currentYear)
	}

	if err != nil {
		logs.Error("Failed to get next control number", map[string]interface{}{
			"dteType":  dteType,
			"branchID": branchID,
			"error":    err.Error(),
		})
		return 0, err
	}

	return next, nil
}

func (r *ControlNumberRepository) nextInTransaction(ctx context.Context, dteType string, branchID uint, year int) (int, error) {
	var sequence db_models.ControlNumberSequence

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("branch_id = ? AND dte_type = ? AND year = ?", branchID, dteType, year).
			First(&sequence)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			sequence = db_models.ControlNumberSequence{
				BranchID:   branchID,
				DTEType:    dteType,
				LastNumber: 1,
				Year:       year,
			}
			return tx.Create(&sequence).Error
		}

		if result.Error != nil {
			return result.Error
		}

		sequence.LastNumber++
		return tx.Save(&sequence).Error
	})
	if err != nil {
		return 0, err
	}

	return sequence.LastNumber, nil
}
