package dte

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
)

// ContingencyDocument represents a document in contingency state
type ContingencyDocument struct {
	ID              string    `json:"id,omitempty"`
	DocumentID      string    `json:"document_id"`
	BranchID        uint      `json:"branch_id"`
	ContingencyType int8      `json:"contingency_type"`
	Reason          string    `json:"reason"`
	BatchID         *string   `json:"batch_id,omitempty"`
	MHBatchID       *string   `json:"mh_batch_id,omitempty"`
	Observations    *string   `json:"observations,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`

	Document *DTEDetails        `json:"document,omitempty"`
	Branch   *user.BranchOffice `json:"branch,omitempty"`
}
