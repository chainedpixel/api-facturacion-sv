package dte

import "time"

// ControlNumberSequence represents the control number sequence for DTEs
type ControlNumberSequence struct {
	ID         uint      `json:"id,omitempty"`
	BranchID   uint      `json:"branch_id"`
	DTEType    string    `json:"dte_type"`
	Year       uint      `json:"year"`
	LastNumber uint      `json:"last_number"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}
