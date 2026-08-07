package event

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, eventType string, userID, branchID uint, payloadJSON string, occurredAt time.Time) error
}
