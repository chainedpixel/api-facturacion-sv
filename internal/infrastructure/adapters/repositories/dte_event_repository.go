package repositories

import (
	"context"

	"gorm.io/gorm"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
)

type DTEEventRepository struct {
	db *gorm.DB
}

func NewDTEEventRepository(db *gorm.DB) dte_documents.DTEEventRepositoryPort {
	return &DTEEventRepository{db: db}
}

func (r *DTEEventRepository) Create(ctx context.Context, event *dte.DTEEvent) error {
	model := &db_models.DTEEvent{
		GenerationCode:   event.GenerationCode,
		BranchID:         event.BranchID,
		EventType:        event.EventType,
		RelatedDTEID:     event.RelatedDTEID,
		ReplacementDTEID: event.ReplacementDTEID,
		JSONData:         event.JSONData,
		SignedDocument:   optionalString(event.SignedDocument),
		ReceptionStamp:   event.ReceptionStamp,
		ProcessingDate:   optionalString(event.ProcessingDate),
		HaciendaResponse: optionalString(event.HaciendaResponse),
		Status:           event.Status,
		CreatedAt:        utils.TimeNow(),
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *DTEEventRepository) ListByRelatedDTE(ctx context.Context, branchID uint, relatedDTEID string) ([]dte.DTEEvent, error) {
	var models []db_models.DTEEvent
	if err := r.db.WithContext(ctx).
		Where("branch_id = ? AND related_dte_id = ?", branchID, relatedDTEID).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, handleGormErr(err, "ListByRelatedDTE")
	}

	events := make([]dte.DTEEvent, 0, len(models))
	for _, m := range models {
		events = append(events, dte.DTEEvent{
			GenerationCode:   m.GenerationCode,
			BranchID:         m.BranchID,
			EventType:        m.EventType,
			RelatedDTEID:     m.RelatedDTEID,
			ReplacementDTEID: m.ReplacementDTEID,
			JSONData:         m.JSONData,
			SignedDocument:   valueOrEmpty(m.SignedDocument),
			ReceptionStamp:   m.ReceptionStamp,
			ProcessingDate:   valueOrEmpty(m.ProcessingDate),
			HaciendaResponse: valueOrEmpty(m.HaciendaResponse),
			Status:           m.Status,
			CreatedAt:        m.CreatedAt,
		})
	}
	return events, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
