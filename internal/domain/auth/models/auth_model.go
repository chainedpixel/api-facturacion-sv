package models

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
)

// AuthCredentials represents the authentication credentials
type AuthCredentials struct {
	MHCredentials *HaciendaCredentials `json:"credentials"`
	APIKey        string               `json:"api_key"`
	APISecret     string               `json:"api_secret"`
}

func (a *AuthCredentials) Validate() error {
	if a.APIKey == "" {
		return dte_errors.NewValidationError("RequiredField", "api_key")
	}
	if a.APISecret == "" {
		return dte_errors.NewValidationError("RequiredField", "api_secret")
	}

	if a.MHCredentials == nil {
		return dte_errors.NewValidationError("RequiredField", "credentials")
	}

	if a.MHCredentials.Username == "" {
		return dte_errors.NewValidationError("RequiredField", "username")
	}

	if a.MHCredentials.Password == "" {
		return dte_errors.NewValidationError("RequiredField", "password")
	}

	return nil
}

// AuthClaims represents the information that will be included in the JWT token
type AuthClaims struct {
	ClientID  uint      `json:"sub"`
	BranchID  uint      `json:"branch_sub"`
	AuthType  string    `json:"auth_type"`
	NIT       string    `json:"nit"`
	ExpiresAt time.Time `json:"expires_at"`
}

// HaciendaCredentials represents the Hacienda credentials
type HaciendaCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
