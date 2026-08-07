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
}

func NewRedisCooldown(client *redis.Client, ttl time.Duration) *RedisCooldown {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &RedisCooldown{client: client, ttl: ttl}
}

func (c *RedisCooldown) ShouldNotify(ctx context.Context, eventName, aggregateID string) (bool, error) {
	key := fmt.Sprintf("notify:cooldown:%s:%s", eventName, aggregateID)
	ok, err := c.client.SetNX(ctx, key, "1", c.ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}
