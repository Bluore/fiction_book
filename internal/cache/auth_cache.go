package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	keyEmailVerify  = "GSF:auth:email:verify:%s:code"
	keyRefreshToken = "GSF:auth:refresh:token:%s"
)

func KeyEmailVerify(key string) string {
	return fmt.Sprintf(keyEmailVerify, key)
}

func KeyRefreshToken(userID string) string {
	return fmt.Sprintf(keyRefreshToken, userID)
}

type AuthCache interface {
	UpdateEmailVerification(ctx context.Context, email, key string, exp time.Duration) error
	GetEmailVerification(ctx context.Context, key string) (string, error)
	DeleteEmailVerification(ctx context.Context, key string) error

	// UpdateRefreshToken 写入 RefreshToken 白名单，同一用户仅保留最新一条，实现单点登录
	UpdateRefreshToken(ctx context.Context, userID, token string, exp time.Duration) error
	// GetRefreshToken 读取白名单中的 RefreshToken，第二个返回值表示是否存在
	GetRefreshToken(ctx context.Context, userID string) (string, bool, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
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

func (c *authCache) UpdateRefreshToken(ctx context.Context, userID, token string, exp time.Duration) error {
	return c.redis.Set(ctx, KeyRefreshToken(userID), token, exp).Err()
}

func (c *authCache) GetRefreshToken(ctx context.Context, userID string) (string, bool, error) {
	token, err := c.redis.Get(ctx, KeyRefreshToken(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return token, true, nil
}

func (c *authCache) DeleteRefreshToken(ctx context.Context, userID string) error {
	return c.redis.Del(ctx, KeyRefreshToken(userID)).Err()
}
