package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	keyEmailVerify = "GSF:auth:email:verify:%s:code"
)

func KeyEmailVerify(key string) string {
	return fmt.Sprintf(keyEmailVerify, key)
}

type AuthCache interface {
	UpdateEmailVerification(ctx context.Context, email, key string, exp time.Duration) error
	GetEmailVerification(ctx context.Context, key string) (string, error)
	DeleteEmailVerification(ctx context.Context, key string) error
}

type authCache struct {
	redis *redis.Client
}

func NewAuthCache(redis *redis.Client) AuthCache {
	return &authCache{redis: redis}
}

func (c *authCache) UpdateEmailVerification(ctx context.Context, email, key string, exp time.Duration) error {
	return c.redis.Set(ctx, KeyEmailVerify(email), key, exp).Err()
}

func (c *authCache) GetEmailVerification(ctx context.Context, key string) (string, error) {
	return c.redis.Get(ctx, KeyEmailVerify(key)).Result()
}

func (c *authCache) DeleteEmailVerification(ctx context.Context, key string) error {
	return c.redis.Del(ctx, KeyEmailVerify(key)).Err()
}
