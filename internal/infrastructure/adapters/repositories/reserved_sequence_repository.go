package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type ReservedSequenceRepository struct {
	db *gorm.DB
}

func NewReservedSequenceRepository(db *gorm.DB) dte_documents.ReservedSequenceRepositoryPort {
	return &ReservedSequenceRepository{db: db}
}

func (r *ReservedSequenceRepository) Create(ctx context.Context, reservation *dte_documents.ReservedSequence) (uint, error) {
	dbModel := &db_models.ReservedSequenceNumber{
		BranchID:              reservation.BranchID,
		DTEType:               reservation.DTEType,
		SequenceNumber:        reservation.SequenceNumber,
		Year:                  reservation.Year,
		Status:                reservation.Status,
		DocumentID:            reservation.DocumentID,
		ReservedAt:            reservation.ReservedAt,
		ExpiresAt:             reservation.ExpiresAt,
		IsContingency:         reservation.IsContingency,
		ContingencyDocumentID: reservation.ContingencyDocumentID,
	}

	if err := r.db.WithContext(ctx).Create(dbModel).Error; err != nil {
		return 0, err
	}

	return dbModel.ID, nil
}

func (r *ReservedSequenceRepository) GetOldestReleasedNumber(ctx context.Context, branchID uint, dteType string, year int) (*dte_documents.ReservedSequence, error) {
	var dbModel db_models.ReservedSequenceNumber

	logs.Debug("Searching for released sequence numbers", map[string]interface{}{
		"branchID": branchID,
		"dteType":  dteType,
		"year":     year,
	})

	err := r.db.WithContext(ctx).
		Where("branch_id = ?", branchID).
		Where("dte_type = ?", dteType).
		Where("year = ?", year).
		Where("status = ?", dte_documents.ReservationStatusReleased).
		Where("(hacienda_code IS NULL OR hacienda_code = '')").
		Order("released_at ASC").
		First(&dbModel).Error

	if err == gorm.ErrRecordNotFound {
		logs.Debug("No released sequence numbers found", map[string]interface{}{
			"branchID": branchID,
			"dteType":  dteType,
			"year":     year,
		})
		return nil, nil
	}

	if err != nil {
		logs.Error("Error searching for released sequence numbers", map[string]interface{}{
			"error":    err.Error(),
			"branchID": branchID,
			"dteType":  dteType,
			"year":     year,
		})
		return nil, err
	}

	logs.Info("Found released sequence number", map[string]interface{}{
		"id":             dbModel.ID,
		"sequenceNumber": dbModel.SequenceNumber,
		"branchID":       dbModel.BranchID,
		"dteType":        dbModel.DTEType,
		"releasedAt":     dbModel.ReleasedAt,
	})

	return r.toDomain(&dbModel), nil
}

func (r *ReservedSequenceRepository) MarkAsReserved(ctx context.Context, reservationID uint, expiresAt *time.Time) error {
	updates := map[string]interface{}{
		"status":           dte_documents.ReservationStatusReserved,
		"reserved_at":      utils.TimeNow(),
		"expires_at":       expiresAt,
		"released_at":      nil,
		"rejection_reason": nil,
		"hacienda_code":    nil,
	}

	result := r.db.WithContext(ctx).
		Model(&db_models.ReservedSequenceNumber{}).
		Where("id = ? AND status = ?", reservationID, dte_documents.ReservationStatusReleased).
		Updates(updates)

	if result.Error != nil {
		logs.Error("Failed to mark reservation as reserved", map[string]interface{}{
			"error":         result.Error.Error(),
			"reservationID": reservationID,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	logs.Info("Marked reservation as reserved", map[string]interface{}{
		"reservationID": reservationID,
		"rowsAffected":  result.RowsAffected,
	})

	return nil
}

func (r *ReservedSequenceRepository) UpdateStatus(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, status string, timestamp time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == dte_documents.ReservationStatusConfirmed {
		updates["confirmed_at"] = timestamp
	} else if status == dte_documents.ReservationStatusReleased {
		updates["released_at"] = timestamp
	}

	query := r.db.WithContext(ctx).Model(&db_models.ReservedSequenceNumber{})

	if branchID != 0 {
		query = query.Where("branch_id = ?", branchID)
	}

	return query.
		Where("dte_type = ?", dteType).
		Where("sequence_number = ?", seqNum).
		Where("year = ?", year).
		Updates(updates).Error
}

func (r *ReservedSequenceRepository) UpdateStatusWithReason(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, status string, timestamp time.Time, reason string, haciendaCode string) error {
	updates := map[string]interface{}{
		"status":           status,
		"rejection_reason": reason,
		"hacienda_code":    haciendaCode,
	}

	if status == dte_documents.ReservationStatusConfirmed {
		updates["confirmed_at"] = timestamp
	} else if status == dte_documents.ReservationStatusReleased {
		updates["released_at"] = timestamp
		updates["is_contingency"] = false
		updates["contingency_document_id"] = nil
	}

	query := r.db.WithContext(ctx).Model(&db_models.ReservedSequenceNumber{})

	if branchID != 0 {
		query = query.Where("branch_id = ?", branchID)
	}

	result := query.
		Where("dte_type = ?", dteType).
		Where("sequence_number = ?", seqNum).
		Where("year = ?", year).
		Updates(updates)

	if result.Error != nil {
		logs.Error("Failed to update reservation status", map[string]interface{}{
			"error":      result.Error.Error(),
			"branchID":   branchID,
			"dteType":    dteType,
			"seqNum":     seqNum,
			"year":       year,
			"status":     status,
			"rowsAffect": result.RowsAffected,
		})
		return result.Error
	}

	logData := map[string]interface{}{
		"branchID":        branchID,
		"dteType":         dteType,
		"seqNum":          seqNum,
		"year":            year,
		"status":          status,
		"rowsAffected":    result.RowsAffected,
		"rejectionReason": reason,
		"haciendaCode":    haciendaCode,
	}

	if status == dte_documents.ReservationStatusReleased {
		logData["contingencyCleared"] = true
	}

	logs.Info("Reservation status updated successfully", logData)

	if result.RowsAffected == 0 {
		logs.Warn("No rows affected when updating reservation status", map[string]interface{}{
			"branchID": branchID,
			"dteType":  dteType,
			"seqNum":   seqNum,
			"year":     year,
		})
	}

	return nil
}

func (r *ReservedSequenceRepository) UpdateStatusWithDocumentID(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, status string, timestamp time.Time, documentID string) error {
	updates := map[string]interface{}{
		"status":      status,
		"document_id": documentID,
	}

	if status == dte_documents.ReservationStatusConfirmed {
		updates["confirmed_at"] = timestamp
	} else if status == dte_documents.ReservationStatusReleased {
		updates["released_at"] = timestamp
	}

	query := r.db.WithContext(ctx).Model(&db_models.ReservedSequenceNumber{})

	if branchID != 0 {
		query = query.Where("branch_id = ?", branchID)
	}

	result := query.
		Where("dte_type = ?", dteType).
		Where("sequence_number = ?", seqNum).
		Where("year = ?", year).
		Updates(updates)

	if result.Error != nil {
		logs.Error("Failed to update reservation status with document ID", map[string]interface{}{
			"error":      result.Error.Error(),
			"branchID":   branchID,
			"dteType":    dteType,
			"seqNum":     seqNum,
			"year":       year,
			"status":     status,
			"documentID": documentID,
		})
		return result.Error
	}

	logs.Info("Reservation status updated with document ID", map[string]interface{}{
		"branchID":     branchID,
		"dteType":      dteType,
		"seqNum":       seqNum,
		"year":         year,
		"status":       status,
		"documentID":   documentID,
		"rowsAffected": result.RowsAffected,
	})

	if result.RowsAffected == 0 {
		logs.Warn("No rows affected when updating reservation status", map[string]interface{}{
			"branchID": branchID,
			"dteType":  dteType,
			"seqNum":   seqNum,
			"year":     year,
		})
	}

	return nil
}

func (r *ReservedSequenceRepository) UpdateAsContingency(ctx context.Context, reservationID uint, documentID string, contingencyDocID string) error {
	updates := map[string]interface{}{
		"document_id":             documentID,
		"is_contingency":          true,
		"contingency_document_id": contingencyDocID,
		"expires_at":              nil,
	}

	return r.db.WithContext(ctx).
		Model(&db_models.ReservedSequenceNumber{}).
		Where("id = ?", reservationID).
		Updates(updates).Error
}

func (r *ReservedSequenceRepository) UpdateAsContingencyByControlNumber(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, documentID string, contingencyDocID string) error {
	updates := map[string]interface{}{
		"document_id":             documentID,
		"is_contingency":          true,
		"contingency_document_id": contingencyDocID,
		"expires_at":              nil,
	}

	query := r.db.WithContext(ctx).Model(&db_models.ReservedSequenceNumber{})

	if branchID != 0 {
		query = query.Where("branch_id = ?", branchID)
	}

	result := query.
		Where("dte_type = ?", dteType).
		Where("sequence_number = ?", seqNum).
		Where("year = ?", year).
		Updates(updates)

	if result.Error != nil {
		logs.Error("Failed to update reservation as contingency", map[string]interface{}{
			"error":            result.Error.Error(),
			"branchID":         branchID,
			"dteType":          dteType,
			"seqNum":           seqNum,
			"year":             year,
			"documentID":       documentID,
			"contingencyDocID": contingencyDocID,
		})
		return result.Error
	}

	logs.Info("Reservation marked as contingency successfully", map[string]interface{}{
		"branchID":         branchID,
		"dteType":          dteType,
		"seqNum":           seqNum,
		"year":             year,
		"documentID":       documentID,
		"contingencyDocID": contingencyDocID,
		"rowsAffected":     result.RowsAffected,
	})

	if result.RowsAffected == 0 {
		logs.Warn("No rows affected when marking reservation as contingency", map[string]interface{}{
			"branchID": branchID,
			"dteType":  dteType,
			"seqNum":   seqNum,
			"year":     year,
		})
	}

	return nil
}

func (r *ReservedSequenceRepository) GetByDocumentID(ctx context.Context, documentID string) (*dte_documents.ReservedSequence, error) {
	var dbModel db_models.ReservedSequenceNumber

	err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		First(&dbModel).Error

	if err != nil {
		return nil, err
	}

	return r.toDomain(&dbModel), nil
}

func (r *ReservedSequenceRepository) GetExpiredNonContingencyReservations(ctx context.Context) ([]dte_documents.ReservedSequence, error) {
	var dbModels []db_models.ReservedSequenceNumber

	err := r.db.WithContext(ctx).
		Where("status = ?", dte_documents.ReservationStatusReserved).
		Where("is_contingency = ?", false).
		Where("expires_at IS NOT NULL").
		Where("expires_at < ?", utils.TimeNow()).
		Find(&dbModels).Error

	if err != nil {
		return nil, err
	}

	result := make([]dte_documents.ReservedSequence, len(dbModels))
	for i, dbModel := range dbModels {
		result[i] = *r.toDomain(&dbModel)
	}

	return result, nil
}

func (r *ReservedSequenceRepository) toDomain(dbModel *db_models.ReservedSequenceNumber) *dte_documents.ReservedSequence {
	return &dte_documents.ReservedSequence{
		ID:                    dbModel.ID,
		BranchID:              dbModel.BranchID,
		DTEType:               dbModel.DTEType,
		SequenceNumber:        dbModel.SequenceNumber,
		Year:                  dbModel.Year,
		Status:                dbModel.Status,
		DocumentID:            dbModel.DocumentID,
		ReservedAt:            dbModel.ReservedAt,
		ConfirmedAt:           dbModel.ConfirmedAt,
		ReleasedAt:            dbModel.ReleasedAt,
		ExpiresAt:             dbModel.ExpiresAt,
		RejectionReason:       dbModel.RejectionReason,
		HaciendaCode:          dbModel.HaciendaCode,
		IsContingency:         dbModel.IsContingency,
		ContingencyDocumentID: dbModel.ContingencyDocumentID,
	}
}
