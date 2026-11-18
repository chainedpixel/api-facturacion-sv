package email

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cooldown interface {
	ShouldNotify(ctx context.Context, eventName, aggregateID string) (bool, error)
}

type RedisCooldown struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisCooldown builds a cooldown backed by Redis; keyPrefix namespaces every key it writes.
func NewRedisCooldown(client *redis.Client, ttl time.Duration, keyPrefix string) *RedisCooldown {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &RedisCooldown{client: client, ttl: ttl, prefix: keyPrefix}
}

func (c *RedisCooldown) ShouldNotify(ctx context.Context, eventName, aggregateID string) (bool, error) {
	key := fmt.Sprintf("%snotify:cooldown:%s:%s", c.prefix, eventName, aggregateID)
	ok, err := c.client.SetNX(ctx, key, "1", c.ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}
