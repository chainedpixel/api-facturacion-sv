package strategies

import (
	"context"
	"errors"
	"strings"

	errPackage "github.com/chainedpixel/ordo-factus/internal/infrastructure/error"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type AuthService struct {
	strategies   map[string]auth.AuthStrategy
	authRepo     auth.AuthRepositoryPort
	tokenService ports.TokenManager
	cacheService ports.CacheManager
}

func NewAuthService(
	tokenService ports.TokenManager,
	clientRepository auth.AuthRepositoryPort,
	cacheService ports.CacheManager,
) auth.AuthManager {
	return &AuthService{
		strategies: map[string]auth.AuthStrategy{
			constants.StandardAuthType: NewStandardAuthStrategy(clientRepository, cacheService),
		},
		tokenService: tokenService,
		authRepo:     clientRepository,
		cacheService: cacheService,
	}
}

// Login handles the authentication process
func (s *AuthService) Login(ctx context.Context, credentials *models.AuthCredentials) (string, error) {
	if !credentialsExists(credentials) {
		return "", shared_error.NewFormattedGeneralServiceError("AuthService", "Login", "MissingCredentials")
	}

	authType, err := s.authRepo.GetAuthTypeByApiKey(ctx, credentials.APIKey)
	if err != nil {
		if errors.Is(err, errPackage.ErrUserNotFound) {
			return "", shared_error.NewFormattedGeneralServiceError("AuthService", "Login", "NotFound")
		}

		return "", err
	}

	strategy, exists := s.strategies[authType]
	if !exists {
		logs.Error("Auth strategy not found", map[string]interface{}{
			"authType": authType,
		})
		return "", shared_error.NewFormattedGeneralServiceError("AuthService", "Login", "ServerError", authType)
	}

	if err = strategy.ValidateCredentials(credentials); err != nil {
		return "", err
	}

	claims, err := strategy.Authenticate(ctx, credentials)
	if err != nil {
		return "", err
	}

	tokenLifetime, err := strategy.GetTokenLifetime(credentials)
	if err != nil {
		return "", err
	}

	token, err := s.tokenService.GenerateToken(claims, tokenLifetime)
	if err != nil {
		return "", err
	}

	if err = s.cacheService.SetCredentials(token, credentials.MHCredentials, tokenLifetime); err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) GetHaciendaCredentials(ctx context.Context, nit, token string) (*models.HaciendaCredentials, error) {

	authType, err := s.authRepo.GetAuthTypeByNIT(ctx, nit)
	if err != nil {
		logs.Info("Error getting auth type", map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	logs.Info("Auth type retrieved", map[string]interface{}{"authType": authType})

	strategy, exists := s.strategies[authType]
	if !exists {
		logs.Info("Unsupported authentication type", map[string]interface{}{"authType": authType})
		return nil, errors.New("unsupported authentication type")
	}
	logs.Info("Strategy found", map[string]interface{}{"strategy": strategy.GetAuthType()})

	return strategy.GetHaciendaCredentials(token)
}

// GetIssuer returns the issuer by its branch ID
func (s *AuthService) GetIssuer(ctx context.Context, branchID uint) (*dte.IssuerDTE, error) {
	return s.authRepo.GetIssuerInfoByBranchID(ctx, branchID)
}

// ValidateToken validates an existing token
func (s *AuthService) ValidateToken(token string) (*models.AuthClaims, error) {
	return s.tokenService.ValidateToken(token)
}

// RevokeToken revokes a token
func (s *AuthService) RevokeToken(token string) error {
	return s.tokenService.RevokeToken(token)
}

// credentialsExists verifies that the credentials have all required fields
func credentialsExists(credentials *models.AuthCredentials) bool {
	return credentials.APIKey != "" && credentials.APISecret != "" && credentials.MHCredentials != nil && credentials.MHCredentials.Username != "" && credentials.MHCredentials.Password != ""
}

// Create creates a user with their branch offices
func (s *AuthService) Create(ctx context.Context, user *user.User) error {
	err := s.authRepo.Create(ctx, user)
	if err != nil {
		return handleGormError("Create", err)
	}

	return nil
}

func (s *AuthService) GetByNIT(ctx context.Context, nit string) (*user.User, error) {
	user, err := s.authRepo.GetByNIT(ctx, nit)
	if err != nil {
		return nil, handleGormError("GetByNIT", err)
	}

	return user, nil
}

func (s *AuthService) GetBranchByBranchID(ctx context.Context, branchID uint) (*user.BranchOffice, error) {
	branch, err := s.authRepo.GetBranchByBranchID(ctx, branchID)
	if err != nil {
		return nil, handleGormError("GetBranchByBranchID", err)
	}

	return branch, nil
}

func handleGormError(operation string, err error) error {
	if errors.Is(err, gorm.ErrInvalidData) {
		return shared_error.NewFormattedGeneralServiceError("AuthService", operation, "InvalidData")
	}

	if isDuplicatedEntryErr(err) {
		errMsg := err.Error()
		if strings.Contains(errMsg, "nit") {
			return shared_error.NewFormattedGeneralServiceError("AuthService", operation, "DuplicatedEntry", "nit")
		}

		if strings.Contains(errMsg, "email") {
			return shared_error.NewFormattedGeneralServiceError("AuthService", operation, "DuplicatedEntry", "email")
		}

		if strings.Contains(errMsg, "phone") {
			return shared_error.NewFormattedGeneralServiceError("AuthService", operation, "DuplicatedEntry", "phone")
		}

		if strings.Contains(errMsg, "nrc") {
			return shared_error.NewFormattedGeneralServiceError("AuthService", operation, "DuplicatedEntry", "nrc")
		}
	}

	return err
}

func isDuplicatedEntryErr(err error) bool {
	errMsg := strings.ToLower(err.Error())
	return errors.Is(err, gorm.ErrInvalidData) ||
		strings.Contains(errMsg, "duplicate entry") ||
		strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "violates unique") ||
		strings.Contains(errMsg, "unique key constraint")
}
