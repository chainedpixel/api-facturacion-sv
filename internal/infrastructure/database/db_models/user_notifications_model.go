package db_models

import "time"

// UserNotification represents the table that will store information about notifications to be sent to users.
// This table will store the information of the notifications that will be sent to users.
type UserNotification struct {
	ID               uint      `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	UserID           uint      `gorm:"column:user_id;type:uint;not null;index:idx_notification_user"`
	EventID          uint      `gorm:"column:event_id;type:uint;not null;index:idx_notification_event"`
	NotificationType string    `gorm:"column:notification_type;type:varchar(15);not null;index"`
	Message          string    `gorm:"column:message;type:text;not null"`
	DeliveryStatus   string    `gorm:"column:delivery_status;type:varchar(15);not null;index"`
	DeliveryAt       time.Time `gorm:"column:delivery_at;type:timestamp;default:CURRENT_TIMESTAMP"`

	User  *User        `gorm:"foreignKey:UserID;references:ID"`
	Event *DomainEvent `gorm:"foreignKey:EventID;references:ID"`
}

func (UserNotification) TableName() string {
	return "user_notifications"
}
