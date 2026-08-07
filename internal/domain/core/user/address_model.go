package user

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
)

// Address represents the address of a branch office or headquarters
type Address struct {
	ID           uint   `json:"-"`
	BranchID     uint   `json:"-"`
	Municipality string `json:"municipality"`
	Department   string `json:"department"`
	Complement   string `json:"complement"`
}

func (a *Address) Validate() error {
	if a.Municipality == "" {
		return dte_errors.NewValidationError("RequiredField", "municipality")
	}
	if a.Department == "" {
		return dte_errors.NewValidationError("RequiredField", "department")
	}

	if a.Complement == "" {
		return dte_errors.NewValidationError("RequiredField", "complement")
	}

	return nil
}
