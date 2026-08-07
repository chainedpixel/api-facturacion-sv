package user

import (
	"encoding/json"
	"time"

	errPackage "github.com/chainedpixel/ordo-factus/internal/domain/core/error"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
)

type User struct {
	ID                   uint      `json:"-"`
	Status               bool      `json:"-"`
	NIT                  string    `json:"nit"`
	NRC                  string    `json:"nrc"`
	AuthType             string    `json:"auth_type"`
	PasswordPri          string    `json:"password_pri"`
	CommercialName       string    `json:"commercial_name"`
	Business             string    `json:"business_name"`
	EconomicActivity     string    `json:"economic_activity"`
	EconomicActivityDesc string    `json:"economic_activity_desc"`
	Email                string    `json:"email"`
	Phone                string    `json:"phone"`
	YearInDTE            bool      `json:"year_in_dte"`
	TokenLifetime        int       `json:"token_lifetime"`
	CreatedAt            time.Time `json:"-"`
	UpdatedAt            time.Time `json:"-"`

	BranchOffices []BranchOffice `json:"branch_offices,omitempty"`
}

// Validate validates the user fields to ensure they comply with business rules
func (u *User) Validate() error {
	if _, err := identification.NewNIT(u.NIT); err != nil {
		return err
	}

	if _, err := identification.NewNRC(u.NRC); err != nil {
		return err
	}

	if _, err := identification.NewActivityCode(u.EconomicActivity); err != nil {
		return err
	}

	if _, err := base.NewPhone(u.Phone); err != nil {
		return err
	}

	if _, err := base.NewEmail(u.Email); err != nil {
		return err
	}

	if u.AuthType == "" {
		return dte_errors.NewValidationError("RequiredField", "auth_type")
	}

	if u.PasswordPri == "" {
		return dte_errors.NewValidationError("RequiredField", "password_pri")
	}

	if u.CommercialName == "" {
		if len(u.CommercialName) > 150 {
			return dte_errors.NewValidationError("InvalidLength", "commercial_name", "1 to 150", u.CommercialName)
		}
		return dte_errors.NewValidationError("RequiredField", "commercial_name")
	}

	if u.Business == "" {
		if len(u.Business) > 200 {
			return dte_errors.NewValidationError("InvalidLength", "business_name", "1 to 200", u.Business)
		}

		return dte_errors.NewValidationError("RequiredField", "business_name")
	}

	if u.EconomicActivityDesc == "" {
		if len(u.EconomicActivityDesc) > 150 {
			return dte_errors.NewValidationError("InvalidLength", "economic_activity_desc", "1 to 150", u.EconomicActivityDesc)
		}

		return dte_errors.NewValidationError("RequiredField", "economic_activity_desc")
	}

	if u.BranchOffices == nil {
		return dte_errors.NewValidationError("RequiredField", "branch_offices")
	}

	if u.TokenLifetime < 0 {
		return dte_errors.NewValidationError("InvalidValue", u.TokenLifetime, "greater than 0", "token_lifetime")
	}

	if err := u.ValidateBranchOffices(); err != nil {
		return err
	}

	return nil
}

// GetBranchOfficeMatrix returns the branch office that is the headquarters
func (u *User) GetBranchOfficeMatrix() (*BranchOffice, error) {
	for _, branchOffice := range u.BranchOffices {
		if branchOffice.EstablishmentType == constants.CasaMatriz {
			return &branchOffice, nil
		}
	}

	return nil, dte_errors.NewFormattedValidationError(errPackage.ErrBranchMatrixNotFound)
}

// ValidateBranchOffices validates the user's branch offices to ensure they comply with business rules
func (u *User) ValidateBranchOffices() error {
	var matrixCount int
	var matrixHasAddress bool

	if len(u.BranchOffices) == 0 {
		return dte_errors.NewFormattedValidationError(errPackage.ErrAtLeastOneBranch)
	}

	for _, branchOffice := range u.BranchOffices {
		if err := branchOffice.Validate(); err != nil {
			return err
		}

		if branchOffice.EstablishmentType == constants.CasaMatriz {
			matrixCount++
			if branchOffice.Address != nil {
				matrixHasAddress = true
			}
		}
	}

	if matrixCount == 0 {
		return dte_errors.NewFormattedValidationError(errPackage.ErrDontHaveBranchMatrix)
	}

	if !matrixHasAddress {
		return dte_errors.NewFormattedValidationError(errPackage.ErrBranchMatrixWithoutAddress)
	}

	if matrixCount > 1 {
		return dte_errors.NewFormattedValidationError(errPackage.ErrMoreThanOneBranchMatrix)
	}

	return nil
}

// SetBranchesKeysAndSecrets assigns keys and secrets to the user's branch offices
func (u *User) SetBranchesKeysAndSecrets(keys []string, secrets []string) {
	for i := range u.BranchOffices {
		u.BranchOffices[i].APIKey = keys[i]
		u.BranchOffices[i].APISecret = secrets[i]
		u.BranchOffices[i].IsActive = true
	}
}

func (u *User) ToStringJSON() string {
	jsonUser, _ := json.Marshal(u)
	return string(jsonUser)
}

func (u *User) ListBranches() []ListBranchesResponse {
	var branches []ListBranchesResponse

	for i, branch := range u.BranchOffices {
		branches = append(branches, ListBranchesResponse{
			BranchNumber:      i + 1,
			EstablishmentType: branch.EstablishmentType,
			EstablishmentCode: branch.EstablishmentCode,
			APIKey:            branch.APIKey,
			APISecret:         branch.APISecret,
		})
	}

	return branches
}
