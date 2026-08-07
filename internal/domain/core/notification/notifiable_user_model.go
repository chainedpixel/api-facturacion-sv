package notification

// NotifiableUser represents a user who can receive notifications
type NotifiableUser struct {
	ID          uint   `json:"id,omitempty"`
	UserID      uint   `json:"user_id"`
	EntityType  string `json:"entity_type"`
	Email       string `json:"email"`
	EnabledPush bool   `json:"enabled_push"`
}
