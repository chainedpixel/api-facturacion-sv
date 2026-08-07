package strategies

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type StandardAuthStrategy struct {
	authRepo     auth.AuthRepositoryPort
	cacheService ports.CacheManager
}

// NewStandardAuthStrategy creates an instance of StandardAuthStrategy. Receives a client repository.
func NewStandardAuthStrategy(repo auth.AuthRepositoryPort, cacheService ports.CacheManager) *StandardAuthStrategy {
	return &StandardAuthStrategy{
		cacheService: cacheService,
		authRepo:     repo,
	}
}

// GetAuthType returns the authentication type.
func (s *StandardAuthStrategy) GetAuthType() string {
	return constants.StandardAuthType
}

// ValidateCredentials validates the authentication credentials. Returns an error if the credentials are invalid.
func (s *StandardAuthStrategy) ValidateCredentials(credentials *models.AuthCredentials) error {
	if credentials.APIKey == "" {
		logs.Error("API key is required", map[string]interface{}{
			"credentials": credentials,
		})
		return shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"ValidateCredentials",
			"RequiredField",
			"api key",
		)
	}
	if credentials.APISecret == "" {
		logs.Error("API secret is required", map[string]interface{}{
			"credentials": credentials,
		})
		return shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"ValidateCredentials",
			"RequiredField",
			"api secret",
		)
	}
	return nil
}

// Authenticate authenticates a client. Returns the claims of the authenticated client.
func (s *StandardAuthStrategy) Authenticate(ctx context.Context, credentials *models.AuthCredentials) (*models.AuthClaims, error) {
	branch, err := s.authRepo.GetBranchByBranchApiKey(ctx, credentials.APIKey)
	if err != nil {
		logs.Error("Invalid credentials", map[string]interface{}{
			"apiKey": credentials.APIKey,
			"error":  err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"Authenticate",
			"NotFound",
		)
	}

	if subtle.ConstantTimeCompare([]byte(credentials.APISecret), []byte(branch.APISecret)) != 1 {
		logs.Error("Invalid credentials", map[string]interface{}{
			"apiKey": credentials.APIKey,
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"Authenticate",
			"InvalidCredentials",
		)
	}

	user, err := s.authRepo.GetByBranchApiKey(ctx, credentials.APIKey)
	if err != nil {
		logs.Error("Invalid credentials", map[string]interface{}{
			"apiKey": credentials.APIKey,
			"error":  err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"Authenticate",
			"InvalidCredentials",
		)
	}

	if !user.Status {
		logs.Error("Client account is not active", map[string]interface{}{
			"clientID": user.ID,
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"Authenticate",
			"UserNotActive",
		)
	}

	claims := &models.AuthClaims{
		ClientID: user.ID,
		BranchID: branch.ID,
		AuthType: user.AuthType,
		NIT:      user.NIT,
	}

	logs.Info("Client authenticated successfully", map[string]interface{}{
		"clientID": claims.ClientID,
	})

	return claims, nil
}

func (s *StandardAuthStrategy) GetTokenLifetime(credentials *models.AuthCredentials) (time.Duration, error) {
	user, err := s.authRepo.GetByBranchApiKey(context.Background(), credentials.APIKey)
	if err != nil {
		logs.Error("Failed to get user information", map[string]interface{}{
			"apiKey": credentials.APIKey,
			"error":  err.Error(),
		})
		return 0, shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"GetTokenLifetime",
			"ServerError",
		)
	}

	return time.Duration(user.TokenLifetime) * 24 * time.Hour, nil
}

// GetHaciendaCredentials retrieves the Hacienda credentials. Returns the Hacienda credentials.
func (s *StandardAuthStrategy) GetHaciendaCredentials(token string) (*models.HaciendaCredentials, error) {
	creds, err := s.cacheService.GetCredentials(token)
	if err != nil {
		logs.Error("Failed to get Hacienda credentials", map[string]interface{}{
			"token": token,
			"error": err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"StandardAuth",
			"GetHaciendaCredentials",
			"FailedToGetCredentials",
		)
	}

	logs.Info("Hacienda credentials retrieved successfully", map[string]interface{}{
		"token": token,
	})

	return creds, nil
}
