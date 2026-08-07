package notification

import "time"

// UserNotification represents a notification for a user
type UserNotification struct {
	ID               uint      `json:"id,omitempty"`
	UserID           uint      `json:"user_id"`
	EventID          uint      `json:"event_id"`
	NotificationType string    `json:"notification_type"`
	Message          string    `json:"message"`
	DeliveryStatus   string    `json:"delivery_status"`
	DeliveryAt       time.Time `json:"delivery_at,omitempty"`
}
