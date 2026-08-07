package auth

import (
	"context"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
)

// AuthRepositoryPort defines the behavior that an authentication repository must implement
type AuthRepositoryPort interface {
	GetAuthTypeByApiKey(context.Context, string) (string, error)
	GetAuthTypeByNIT(context.Context, string) (string, error)
	GetByNIT(context.Context, string) (*user.User, error)
	GetIssuerInfoByBranchID(context.Context, uint) (*dte.IssuerDTE, error)
	GetByBranchID(context.Context, uint) (*user.User, error)
	GetBranchByBranchID(context.Context, uint) (*user.BranchOffice, error)
	GetBranchByBranchApiKey(context.Context, string) (*user.BranchOffice, error)
	GetByBranchApiKey(context.Context, string) (*user.User, error)
	Create(context.Context, *user.User) error
	Update(context.Context, *user.User) error
	UpdateBranchOffices(context.Context, uint, []user.BranchOffice) error
	DeleteBranchOffice(context.Context, uint, uint) error
	GetMatrixBranch(context.Context, uint) (*user.BranchOffice, error)
}

// AuthStrategy defines the behavior that each authentication strategy must implement
type AuthStrategy interface {
	GetAuthType() string
	Authenticate(ctx context.Context, credentials *models.AuthCredentials) (*models.AuthClaims, error)
	ValidateCredentials(credentials *models.AuthCredentials) error
	GetHaciendaCredentials(token string) (*models.HaciendaCredentials, error)
	GetTokenLifetime(credentials *models.AuthCredentials) (time.Duration, error)
}

// AuthManager defines the behavior of an authentication service
type AuthManager interface {
	Login(ctx context.Context, credentials *models.AuthCredentials) (string, error)
	GetByNIT(ctx context.Context, nit string) (*user.User, error)
	GetBranchByBranchID(ctx context.Context, branchID uint) (*user.BranchOffice, error)
	GetIssuer(ctx context.Context, branchID uint) (*dte.IssuerDTE, error)
	GetHaciendaCredentials(ctx context.Context, nit, token string) (*models.HaciendaCredentials, error)
	Create(ctx context.Context, user *user.User) error
}
