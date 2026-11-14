package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
)

type ContingencyRepository struct {
	db *gorm.DB
}

func NewContingencyRepository(db *gorm.DB) contingency.ContingencyRepositoryPort {
	return &ContingencyRepository{db: db}
}

// Create stores a contingency document in the database
func (r *ContingencyRepository) Create(ctx context.Context, doc *dte.ContingencyDocument) error {
	id := uuid.NewString()
	contingencyDoc := &db_models.ContingencyDocument{
		ID:              id,
		BranchID:        doc.BranchID,
		DocumentID:      doc.DocumentID,
		ContingencyType: doc.ContingencyType,
		Reason:          doc.Reason,
		CreatedAt:       utils.TimeNow(),
		UpdatedAt:       utils.TimeNow(),
	}

	err := r.db.WithContext(ctx).Create(contingencyDoc).Error
	if err != nil {
		return err
	}

	doc.ID = id
	return nil
}

func (r *ContingencyRepository) GetPending(ctx context.Context, limit int) ([]dte.ContingencyDocument, error) {
	var dbDocs []db_models.ContingencyDocument
	err := r.db.WithContext(ctx).
		Preload("Document").
		Preload("Branch").
		Preload("Branch.User").
		Preload("Branch.Address").
		Joins("JOIN dte_details ON contingency_documents.document_id = dte_details.id").
		Where("dte_details.status = ?", constants.DocumentPending).
		Limit(limit).
		Order("contingency_documents.event_stamped_at IS NULL").
		Order("contingency_documents.event_stamped_at asc").
		Order("contingency_documents.created_at asc").
		Find(&dbDocs).Error
	if err != nil {
		return nil, err
	}

	docs := make([]dte.ContingencyDocument, len(dbDocs))
	for i, doc := range dbDocs {
		docs[i] = convertToDomainModel(&doc)
	}

	return docs, nil
}

func (r *ContingencyRepository) UpdateBatch(ctx context.Context, ids []string, observations []string, results map[string]dte.ReceptionResult, batchID string, mhBatchID string, status string) (returnErr error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if rec := recover(); rec != nil {
			logs.Error("Recovered from panic during contingency batch update", map[string]interface{}{"panic": fmt.Sprint(rec)})
			tx.Rollback()
			returnErr = fmt.Errorf("panic during batch update: %v", rec)
		}
	}()

	for i, id := range ids {
		contingencyUpdate := map[string]interface{}{
			"batch_id":    batchID,
			"mh_batch_id": mhBatchID,
		}

		if len(observations) > i {
			contingencyUpdate["observations"] = observations[i]
		}

		if err := tx.Model(&db_models.ContingencyDocument{}).
			Where("id = ?", id).
			Updates(contingencyUpdate).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update document %s: %w", id, err)
		}

		var contingencyDoc db_models.ContingencyDocument
		if err := tx.Where("id = ?", id).First(&contingencyDoc).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to get document %s: %w", id, err)
		}

		dteUpdate := map[string]interface{}{
			"status": status,
		}

		// Normativa v2.0 (Anexo I, regla 4.1): la estructura firmada no se altera; el sello se guarda aparte
		// y se agrega como selloRecibido al armar el Archivo DTE
		if result, exists := results[id]; exists && result.ReceptionStamp != "" {
			dteUpdate["reception_stamp"] = result.ReceptionStamp
			if result.ProcessingDate != "" {
				dteUpdate["processing_date"] = result.ProcessingDate
			}
		}

		if err := tx.Model(&db_models.DTEDetails{}).
			Where("id = ?", contingencyDoc.DocumentID).
			Updates(dteUpdate).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update document %s: %w", id, err)
		}
	}

	return tx.Commit().Error
}

// MarkEventStamped stores the contingency event stamp date on the given documents, only when it was not set
// before, so a resent event does not restart the transmission window.
func (r *ContingencyRepository) MarkEventStamped(ctx context.Context, ids []string, stampedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&db_models.ContingencyDocument{}).
		Where("id IN ? AND event_stamped_at IS NULL", ids).
		Updates(map[string]interface{}{
			"event_stamped_at": stampedAt,
			"updated_at":       utils.TimeNow(),
		}).Error
}

func (r *ContingencyRepository) GetFirstContingencyTimestamp(ctx context.Context, branchID uint) (*time.Time, error) {
	var doc db_models.ContingencyDocument

	err := r.db.WithContext(ctx).
		Joins("JOIN dte_details ON contingency_documents.document_id = dte_details.id").
		Where("contingency_documents.branch_id = ? AND dte_details.status = ?", branchID, constants.DocumentPending).
		Order("contingency_documents.created_at ASC").
		Limit(1).
		Select("contingency_documents.created_at").
		First(&doc).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting first contingency timestamp: %w", err)
	}

	return &doc.CreatedAt, nil
}

func convertToDomainModel(doc *db_models.ContingencyDocument) dte.ContingencyDocument {
	var document *dte.DTEDetails
	if doc.Document != nil {
		document = &dte.DTEDetails{
			ID:             doc.Document.ID,
			DTEType:        doc.Document.DTEType,
			ControlNumber:  doc.Document.ControlNumber,
			Transmission:   doc.Document.Transmission,
			Status:         doc.Document.Status,
			ReceptionStamp: doc.Document.ReceptionStamp,
			JSONData:       doc.Document.JSONData,
		}
	}

	return dte.ContingencyDocument{
		ID:              doc.ID,
		BranchID:        doc.BranchID,
		DocumentID:      doc.DocumentID,
		ContingencyType: doc.ContingencyType,
		Reason:          doc.Reason,
		EventStampedAt:  doc.EventStampedAt,
		CreatedAt:       doc.CreatedAt,
		Document:        document,
		Branch: &user.BranchOffice{
			User: &user.User{
				ID:                   doc.Branch.User.ID,
				Status:               doc.Branch.User.Status,
				Email:                doc.Branch.User.Email,
				Phone:                doc.Branch.User.Phone,
				NIT:                  doc.Branch.User.NIT,
				NRC:                  doc.Branch.User.NRC,
				AuthType:             doc.Branch.User.AuthType,
				EconomicActivity:     doc.Branch.User.EconomicActivity,
				EconomicActivityDesc: doc.Branch.User.EconomicActivityDesc,
			},
		},
	}
}
