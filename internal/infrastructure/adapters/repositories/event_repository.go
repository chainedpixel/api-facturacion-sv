package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) event.Repository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Save(ctx context.Context, eventType string, userID, branchID uint, payloadJSON string, occurredAt time.Time) error {
	row := &db_models.DomainEvent{
		UserID:     userID,
		BranchID:   branchID,
		EventType:  eventType,
		Payload:    payloadJSON,
		OccurredAt: occurredAt.Format(time.RFC3339),
	}
	return r.db.WithContext(ctx).Create(row).Error
}
