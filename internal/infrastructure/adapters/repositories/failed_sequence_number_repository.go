package repositories

import (
	"context"
	"encoding/json"

	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"gorm.io/gorm"
)

type FailedSequenceNumberRepository struct {
	db *gorm.DB
}

// NewFailedSequenceNumberRepository creates a new instance of FailedSequenceNumberRepository
func NewFailedSequenceNumberRepository(db *gorm.DB) ports.FailedSequenceNumberRepositoryPort {
	return &FailedSequenceNumberRepository{db: db}
}

// RegisterFailedSequence registers a failed sequence number with its details
func (r *FailedSequenceNumberRepository) RegisterFailedSequence(
	ctx context.Context,
	branchID uint,
	dteType string,
	sequenceNumber uint,
	year uint,
	failureReason string,
	responseCode string,
	originalRequestData interface{},
	mhResponse string,
) error {
	requestDataJSON, err := json.Marshal(originalRequestData)
	if err != nil {
		logs.Error("Failed to marshal original request data", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	failedSeq := &db_models.FailedSequenceNumber{
		BranchID:            branchID,
		DTEType:             dteType,
		SequenceNumber:      sequenceNumber,
		Year:                year,
		FailureReason:       failureReason,
		ResponseCode:        responseCode,
		OriginalRequestData: string(requestDataJSON),
		MHResponse:          mhResponse,
		CreatedAt:           utils.TimeNow(),
	}

	result := r.db.WithContext(ctx).Create(failedSeq)
	if result.Error != nil {
		logs.Error("Failed to register failed sequence number", map[string]interface{}{
			"error":          result.Error.Error(),
			"branchID":       branchID,
			"dteType":        dteType,
			"sequenceNumber": sequenceNumber,
		})
		return result.Error
	}

	logs.Info("Failed sequence number registered successfully", map[string]interface{}{
		"id":             failedSeq.ID,
		"branchID":       branchID,
		"dteType":        dteType,
		"sequenceNumber": sequenceNumber,
		"responseCode":   responseCode,
	})

	return nil
}

// GetFailedSequences retrieves a list of failed sequence numbers for a specific branch and DTE type
func (r *FailedSequenceNumberRepository) GetFailedSequences(
	ctx context.Context,
	branchID uint,
	dteType string,
	limit int,
) ([]db_models.FailedSequenceNumber, error) {
	var failedSequences []db_models.FailedSequenceNumber

	result := r.db.WithContext(ctx).
		Where("branch_id = ? AND dte_type = ?", branchID, dteType).
		Order("created_at DESC").
		Limit(limit).
		Find(&failedSequences)

	if result.Error != nil {
		logs.Error("Failed to get failed sequences", map[string]interface{}{
			"error":    result.Error.Error(),
			"branchID": branchID,
			"dteType":  dteType,
		})
		return nil, result.Error
	}

	return failedSequences, nil
}

// GetFailedSequencesByYear retrieves a list of failed sequence numbers for a specific branch, DTE type, and year
func (r *FailedSequenceNumberRepository) GetFailedSequencesByYear(
	ctx context.Context,
	branchID uint,
	dteType string,
	year uint,
	limit int,
) ([]db_models.FailedSequenceNumber, error) {
	var failedSequences []db_models.FailedSequenceNumber

	result := r.db.WithContext(ctx).
		Where("branch_id = ? AND dte_type = ? AND year = ?", branchID, dteType, year).
		Order("created_at DESC").
		Limit(limit).
		Find(&failedSequences)

	if result.Error != nil {
		logs.Error("Failed to get failed sequences by year", map[string]interface{}{
			"error":    result.Error.Error(),
			"branchID": branchID,
			"dteType":  dteType,
			"year":     year,
		})
		return nil, result.Error
	}

	return failedSequences, nil
}
