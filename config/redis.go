package config

import "strings"

// redisKeySeparator is appended to the configured prefix so every key keeps a namespace delimiter.
const redisKeySeparator = ":"

// RedisConfig holds the connection settings and the key namespace used by the Redis adapters.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	Prefix   string
}

// NewRedisConfig builds a RedisConfig from the loaded environment variables.
func NewRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     Redis.Host,
		Port:     Redis.Port,
		Password: Redis.Password,
		Prefix:   Redis.Prefix,
	}
}

// GetURL returns the Redis connection URL, including the password when one is configured.
func (c *RedisConfig) GetURL() string {
	if c.Password != "" {
		return "redis://:" + c.Password + "@" + c.Host + ":" + c.Port
	}
	return "redis://" + c.Host + ":" + c.Port
}

// KeyPrefix returns the normalized namespace prepended to every Redis key.
// An empty prefix keeps keys unchanged; otherwise the value always ends with a single ":" separator.
func (c *RedisConfig) KeyPrefix() string {
	prefix := strings.TrimRight(strings.TrimSpace(c.Prefix), redisKeySeparator)
	if prefix == "" {
		return ""
	}
	return prefix + redisKeySeparator
}
