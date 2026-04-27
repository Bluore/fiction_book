package cache

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	keyUserAssetGold = "Cache:User:Asset:%s:Gold"
	keyUserAsset     = "Cache:User:Asset:%s"
)

func KeyUserAssetGold(userId string) string {
	return fmt.Sprintf(keyUserAssetGold, userId)
}
func KeyUserAsset(userId string) string {
	return fmt.Sprintf(keyUserAsset, userId)
}

type UserAssetCache interface {
	UpdateGold(ctx context.Context, userId string, gold int) error
	GetGold(ctx context.Context, userId string) (int64, bool, error)
	Update(ctx context.Context, userId string, asset *model.UserAsset) error
	Get(ctx context.Context, userId string) (*model.UserAsset, bool, error)
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

func (c userAssetCache) Update(ctx context.Context, userId string, asset *model.UserAsset) error {
	assetEncode, err := json.Marshal(asset)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, KeyUserAsset(userId), assetEncode, time.Hour).Err()
}

func (c userAssetCache) Get(ctx context.Context, userId string) (*model.UserAsset, bool, error) {
	cache, err := c.redis.Get(ctx, KeyUserAsset(userId)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var asset model.UserAsset
	err = json.Unmarshal([]byte(cache), asset)
	if err != nil {
		return nil, false, err
	}
	return &asset, true, nil
}
