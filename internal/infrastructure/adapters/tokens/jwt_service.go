package tokens

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/golang-jwt/jwt/v5"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type JWTService struct {
	SecretKey    string
	cacheService ports.CacheManager
}

// NewJWTService creates an instance of JWTService. Receives a secret key and a cacheService.
func NewJWTService(secretKey string, cache ports.CacheManager) *JWTService {
	return &JWTService{
		SecretKey:    secretKey,
		cacheService: cache,
	}
}

// GenerateToken generates a JWT token with the provided claims.
func (s *JWTService) GenerateToken(claims *models.AuthClaims, tokenLifetime time.Duration) (string, error) {
	now := utils.TimeNow()
	exp := now.Add(tokenLifetime)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        claims.ClientID,
		"branch_sub": claims.BranchID,
		"auth_type":  claims.AuthType,
		"nit":        claims.NIT,
		"exp":        exp.Unix(),
		"iat":        now.Unix(),
	})

	signedToken, err := token.SignedString([]byte(s.SecretKey))
	if err != nil {
		logs.Error("Failed to sign the token", map[string]interface{}{
			"error": err.Error(),
		})
		return "", shared_error.NewGeneralServiceError(
			"JWTService",
			"GenerateToken",
			"failed to sign the token",
			err,
		)
	}

	err = s.SaveTimestampsForContingency(now, exp, tokenLifetime, claims)
	if err != nil {
		logs.Error("Failed to save timestamps for contingency", map[string]interface{}{
			"error": err.Error(),
		})
		return "", shared_error.NewGeneralServiceError(
			"JWTService",
			"GenerateToken",
			"failed to save timestamps for contingency",
			err,
		)
	}

	fmt.Printf("token, %s\n", signedToken)
	key := "token:" + signedToken
	jsonClaims, err := json.Marshal(claims)
	if err != nil {
		logs.Error("Failed to marshal claims", map[string]interface{}{
			"error": err.Error(),
		})
		return "", shared_error.NewGeneralServiceError(
			"JWTService",
			"GenerateToken",
			"failed to marshal claims",
			err,
		)
	}

	err = s.cacheService.Set(key, jsonClaims, tokenLifetime)
	if err != nil {
		logs.Error("Failed to store token in cacheService", map[string]interface{}{
			"error": err.Error(),
		})
		return "", shared_error.NewGeneralServiceError(
			"JWTService",
			"GenerateToken",
			"failed to store token in cacheService",
			err,
		)
	}

	logs.Info("Token generated successfully", map[string]interface{}{
		"clientID": claims.ClientID,
	})

	return signedToken, nil
}

// SaveTimestampsForContingency saves the timestamps of a contingency token.
func (s *JWTService) SaveTimestampsForContingency(issuedAt, expiresAt time.Time, tokenLifetime time.Duration, claims *models.AuthClaims) error {
	timestamps := TokenTimestamps{
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}

	key := fmt.Sprintf("token:timestamps:%d", claims.ClientID)
	jsonTimestamps, err := json.Marshal(timestamps)
	if err != nil {
		logs.Error("Failed to marshal timestamps", map[string]interface{}{
			"error": err.Error(),
		})
		return shared_error.NewGeneralServiceError(
			"JWTService",
			"SaveTimestampsForContingency",
			"failed to marshal timestamps",
			err,
		)
	}

	if err = s.cacheService.Set(key, jsonTimestamps, tokenLifetime); err != nil {
		logs.Error("Failed to store timestamps in cacheService", map[string]interface{}{
			"error": err.Error(),
		})
		return shared_error.NewGeneralServiceError(
			"JWTService",
			"SaveTimestampsForContingency",
			"failed to store timestamps in cacheService",
			err,
		)
	}

	return nil
}

// ValidateToken validates a JWT token and returns the claims if it is valid.
func (s *JWTService) ValidateToken(tokenString string) (*models.AuthClaims, error) {
	key := "token:" + tokenString

	claims, err := s.cacheService.Get(key)
	if err != nil {
		logs.Error("Failed to retrieve token from cacheService", map[string]interface{}{
			"token": tokenString,
			"error": err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceWithError(
			"JWTService",
			"ValidateToken",
			err,
			"FailedToLogin",
		)
	}

	var authClaims models.AuthClaims
	if err = json.Unmarshal([]byte(claims), &authClaims); err != nil {
		logs.Error("Failed to unmarshal claims", map[string]interface{}{
			"token": tokenString,
			"error": err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"JWTService",
			"ValidateToken",
			"AuthServiceUnavailable",
		)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.SecretKey), nil
	})
	if err != nil {
		logs.Error("Failed to parse token", map[string]interface{}{
			"token": tokenString,
			"error": err.Error(),
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"JWTService",
			"ValidateToken",
			"AuthServiceUnavailable",
		)
	}

	if !token.Valid {
		logs.Error("Invalid token", map[string]interface{}{
			"token": tokenString,
		})
		return nil, shared_error.NewFormattedGeneralServiceError(
			"JWTService",
			"ValidateToken",
			"Unauthorized",
		)
	}

	logs.Info("Token validated successfully", map[string]interface{}{
		"token": tokenString,
	})

	return &authClaims, nil
}

// RevokeToken revokes a JWT token.
func (s *JWTService) RevokeToken(token string) error {
	err := s.cacheService.Delete(token)
	if err != nil {
		logs.Error("Failed to revoke token", map[string]interface{}{
			"token": token,
			"error": err.Error(),
		})
		return shared_error.NewGeneralServiceError(
			"JWTService",
			"RevokeToken",
			"failed to revoke token",
			err,
		)
	}

	logs.Info("Token revoked successfully", map[string]interface{}{
		"token": token,
	})

	return nil
}

// GetSecretKey returns the secret key for signing tokens.
func (s *JWTService) GetSecretKey() string {
	return s.SecretKey
}
