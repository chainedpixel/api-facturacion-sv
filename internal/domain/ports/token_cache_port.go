package ports

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/go-redis/redis/v8"
)

// CacheManager interface for cache abstraction
type CacheManager interface {
	Set(key string, claims []byte, ttl time.Duration) error
	SetCredentials(token string, cipherInfo *models.HaciendaCredentials, ttl time.Duration) error
	GetCredentials(token string) (*models.HaciendaCredentials, error)
	Get(key string) (string, error)
	Delete(token string) error
	GetRedisClient() *redis.Client
	CacheListManager
}

// CacheListManager defines the behavior for cache list management
type CacheListManager interface {
	RPush(key string, value []byte) error
	LPush(key string, value []byte) error
	LRange(key string, start, stop int64) ([]string, error)
	LLen(key string) (int64, error)
	LTrim(key string, start, stop int64) error
	Expire(key string, ttl time.Duration) error
	ScanKeys(pattern string) ([]string, error)
}

// TokenManager defines the behavior for token management
type TokenManager interface {
	GenerateToken(claims *models.AuthClaims, tokenLifetime time.Duration) (string, error)
	ValidateToken(token string) (*models.AuthClaims, error)
	RevokeToken(token string) error
	SaveTimestampsForContingency(issuedAt, expiresAt time.Time, tokenLifetime time.Duration, claims *models.AuthClaims) error
	GetSecretKey() string
}
