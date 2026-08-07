package user

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
)

type BranchOffice struct {
	ID                  uint     `json:"-"`
	UserID              uint     `json:"-"`
	EstablishmentCode   *string  `json:"establishment_code,omitempty"`
	EstablishmentCodeMH *string  `json:"establishment_code_mh,omitempty"`
	Email               *string  `json:"email,omitempty"`
	APIKey              string   `json:"api_key"`
	APISecret           string   `json:"api_secret"`
	Phone               *string  `json:"phone,omitempty"`
	EstablishmentType   string   `json:"establishment_type"`
	POSCode             *string  `json:"pos_code,omitempty"`
	POSCodeMH           *string  `json:"pos_code_mh,omitempty"`
	IsActive            bool     `json:"is_active"`
	Address             *Address `json:"address,omitempty"`
	User                *User    `json:"user,omitempty"`
}

func (b *BranchOffice) Validate() error {
	if b.EstablishmentCodeMH != nil {
		if len(*b.EstablishmentCodeMH) != 4 {
			return dte_errors.NewValidationError("InvalidLength", "establishment_code", "4", fmt.Sprint(*b.EstablishmentCodeMH))
		}
	}

	if b.EstablishmentCode != nil {
		if len(*b.EstablishmentCode) < 1 || len(*b.EstablishmentCode) > 10 {
			return dte_errors.NewValidationError("InvalidLength", "establishment_code", "1 to 10", fmt.Sprint(*b.EstablishmentCode))
		}
	}

	if b.POSCodeMH != nil {
		if len(*b.POSCodeMH) != 4 {
			return dte_errors.NewValidationError("InvalidLength", "pos_code_mh", "4", fmt.Sprint(*b.POSCodeMH))
		}
	}

	if b.POSCode != nil {
		if len(*b.POSCode) < 1 || len(*b.POSCode) > 15 {
			return dte_errors.NewValidationError("InvalidLength", "pos_code", "1 to 15", fmt.Sprint(*b.POSCode))
		}
	}

	if b.Email != nil {
		if _, err := base.NewEmail(*b.Email); err != nil {
			return err
		}
	}

	if b.Phone != nil {
		if _, err := base.NewPhone(*b.Phone); err != nil {
			return err
		}
	}

	if _, err := document.NewEstablishmentType(b.EstablishmentType); err != nil {
		return err
	}

	if b.Address != nil {
		if err := b.Address.Validate(); err != nil {
			return err
		}
	}

	return nil
}
