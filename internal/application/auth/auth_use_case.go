package auth

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type AuthUseCase struct {
	authManager  auth.AuthManager
	cryptManager ports.CryptManager
}

func NewAuthUseCase(authManager auth.AuthManager, cryptManager ports.CryptManager) *AuthUseCase {
	return &AuthUseCase{
		authManager:  authManager,
		cryptManager: cryptManager,
	}
}

func (a *AuthUseCase) Login(ctx context.Context, credentials *models.AuthCredentials) (string, error) {
	if err := credentials.Validate(); err != nil {
		return "", err
	}

	token, err := a.authManager.Login(ctx, credentials)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (a *AuthUseCase) Register(ctx context.Context, user *user.User) ([]user.ListBranchesResponse, error) {
	if err := user.Validate(); err != nil {
		return nil, err
	}

	keys, secrets, err := a.cryptManager.GenerateBulkAPIKeys(len(user.BranchOffices))
	if err != nil {
		logs.Error("Failed to generate bulk API keys and secrets", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceError("AuthUseCase", "Register", "FailedToCreateUser")
	}

	user.SetBranchesKeysAndSecrets(keys, secrets)

	if err = a.authManager.Create(ctx, user); err != nil {
		logs.Error("Failed to create user", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceWithError("AuthUseCase", "Register", err, "FailedToCreateUser")
	}

	return user.ListBranches(), nil
}
