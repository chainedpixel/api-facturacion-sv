package db_models

// DomainEvent represents a domain event that is saved in the database.
// Its purpose is to store the domain events generated in the application for specific situations that require
// the attention of users.
// For example, when a branch enters a contingency state, a domain event is generated to notify the user.
type DomainEvent struct {
	ID         uint   `gorm:"column:id;type:uint;primaryKey;autoIncrement;not null"`
	UserID     uint   `gorm:"column:user_id;type:uint;not null;index:idx_event_user"`
	BranchID   uint   `gorm:"column:branch_id;type:uint;not null;index:idx_event_branch"`
	EventType  string `gorm:"column:event_type;type:varchar(50);not null;index"`
	Payload    string `gorm:"column:payload;type:json;not null"`
	OccurredAt string `gorm:"column:occurred_at;type:timestamp;not null;index"`

	User   *User         `gorm:"foreignKey:UserID;references:ID"`
	Branch *BranchOffice `gorm:"foreignKey:BranchID;references:ID"`
}

func (DomainEvent) TableName() string {
	return "domain_events"
}
