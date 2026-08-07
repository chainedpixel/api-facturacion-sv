package contingency

import (
	"encoding/json"
	"fmt"

	"github.com/golang-jwt/jwt/v5"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	authPorts "github.com/chainedpixel/ordo-factus/internal/domain/ports"
)

// contingencyTokenService handles JWT token generation for contingency operations.
// It retrieves cached timestamp data and assembles a signed token matching the original session.
type contingencyTokenService struct {
	cache        authPorts.CacheManager
	tokenService authPorts.TokenManager
	authManager  auth.AuthManager
}

func newContingencyTokenService(
	cache authPorts.CacheManager,
	tokenService authPorts.TokenManager,
	authManager auth.AuthManager,
) *contingencyTokenService {
	return &contingencyTokenService{
		cache:        cache,
		tokenService: tokenService,
		authManager:  authManager,
	}
}

// GenerateTokenForUser generates a signed JWT token for the given user and branch, reusing
// the timestamp stored in cache for the user's original session.
func (s *contingencyTokenService) GenerateTokenForUser(client *user.User, branchID uint) (string, error) {
	key := fmt.Sprintf("token:timestamps:%d", client.ID)
	timestamps, err := s.loadTimestamps(key)
	if err != nil {
		return "", err
	}

	claims := &authModels.AuthClaims{
		ClientID: client.ID,
		AuthType: client.AuthType,
		BranchID: branchID,
		NIT:      client.NIT,
	}

	return s.signToken(claims, timestamps.IssuedAt, timestamps.ExpiresAt)
}

// GenerateTokenForBranch generates a signed JWT token for the given branch office, reusing
// the timestamp stored in cache for the user's original session.
func (s *contingencyTokenService) GenerateTokenForBranch(client *user.BranchOffice) (string, error) {
	key := fmt.Sprintf("token:timestamps:%d", client.User.ID)
	timestamps, err := s.loadTimestamps(key)
	if err != nil {
		return "", err
	}

	claims := &authModels.AuthClaims{
		ClientID: client.User.ID,
		BranchID: client.ID,
		AuthType: client.User.AuthType,
		NIT:      client.User.NIT,
	}

	return s.signToken(claims, timestamps.IssuedAt, timestamps.ExpiresAt)
}

func (s *contingencyTokenService) loadTimestamps(key string) (*tokenTimestamps, error) {
	jsonTimestamps, err := s.cache.Get(key)
	if err != nil {
		return nil, err
	}

	var ts tokenTimestamps
	if err = json.Unmarshal([]byte(jsonTimestamps), &ts); err != nil {
		return nil, err
	}

	return &ts, nil
}

func (s *contingencyTokenService) signToken(claims *authModels.AuthClaims, issuedAt, expiresAt int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        claims.ClientID,
		"branch_sub": claims.BranchID,
		"auth_type":  claims.AuthType,
		"nit":        claims.NIT,
		"exp":        expiresAt,
		"iat":        issuedAt,
	})
	return token.SignedString([]byte(s.tokenService.GetSecretKey()))
}

type tokenTimestamps struct {
	IssuedAt  int64 `json:"IssuedAt"`
	ExpiresAt int64 `json:"ExpiresAt"`
}
