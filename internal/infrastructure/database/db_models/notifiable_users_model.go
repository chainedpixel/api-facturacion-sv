package db_models

// NotifiableUser represents the table whose information will be crucial for the notification process generated through
// domain events. This table will store the information of users who wish to receive notifications.
// EntityID and EntityType indicate that these attributes are polymorphic, meaning they can be either a Client User or an
// Administrator User.
type NotifiableUser struct {
	ID          uint   `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	UserID      uint   `gorm:"column:user_id;type:uint;not null;index:idx_entity,priority:1"`
	EntityType  string `gorm:"column:entity_type;type:varchar(10);not null;index:idx_entity,priority:2"`
	Email       string `gorm:"column:email;type:varchar(255);not null;index:idx_notifiable_email"`
	EnabledPush bool   `gorm:"column:enabled_push;type:tinyint;not null;index:idx_push_enabled"`
}

func (NotifiableUser) TableName() string {
	return "notification_users"
}
