package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	keyUserAssetGold = "Cache:User:Asset:%s:Gold"
)

func KeyUserAssetGold(userId string) string {
	return fmt.Sprintf(keyUserAssetGold, userId)
}

type UserAssetCache interface {
	UpdateGold(ctx context.Context, userId string, gold int) error
	GetGold(ctx context.Context, userId string) (int64, bool, error)
}

type userAssetCache struct {
	redis *redis.Client
}

func NewUserAssetCache(redis *redis.Client) UserAssetCache {
	return &userAssetCache{redis: redis}
}

func (c userAssetCache) UpdateGold(ctx context.Context, userId string, gold int) error {
	return c.redis.Set(ctx, fmt.Sprintf(keyUserStatReadBookNum, userId), gold, time.Hour).Err()
}

func (c userAssetCache) GetGold(ctx context.Context, userId string) (int64, bool, error) {
	cache, err := c.redis.Get(ctx, KeyUserAssetGold(userId)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	res, err := strconv.ParseInt(cache, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return res, true, nil
}
