package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"fiction-book/internal/model"
	"fiction-book/pkg/constant"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	keyUserAssetGold = "GSF:user:asset:user_asset:%s:gold"
	keyUserAsset     = "GSF:user:asset:user_asset:%s:detail"
	keyUserAssetLock = "GSF:user:lock:user_asset:%s"

	timeUserAsset = time.Hour
)

func KeyUserAssetGold(userId string) string {
	return fmt.Sprintf(keyUserAssetGold, userId)
}

func KeyUserAsset(userId string) string {
	return fmt.Sprintf(keyUserAsset, userId)
}

func KeyUserAssetLock(userId string) string {
	return fmt.Sprintf(keyUserAssetLock, userId)
}

func TimeUserAsset() time.Duration {
	return timeUserAsset + TimeShakeDuration(timeUserAsset)
}

type UserAssetCache interface {
	GetUserAsset(ctx context.Context, userId string) (*model.UserAsset, bool, error)
	UpdateUserAsset(ctx context.Context, userId string, asset *model.UserAsset) error
	DeleteUserAsset(ctx context.Context, userId string) (bool, error)

	LockUserAsset(ctx context.Context, userId string) (string, bool, error)
	UnlockUserAsset(ctx context.Context, userId string, token string) error

	UpdateGold(ctx context.Context, userId string, gold int) error
	GetGold(ctx context.Context, userId string) (int64, bool, error)
}

type userAssetCache struct {
	redis *redis.Client
}

func NewUserAssetCache(redis *redis.Client) UserAssetCache {
	return &userAssetCache{redis: redis}
}

func (c userAssetCache) GetUserAsset(ctx context.Context, userId string) (*model.UserAsset, bool, error) {
	value, err := c.redis.Get(ctx, KeyUserAsset(userId)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if value == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var asset model.UserAsset
	if err := json.Unmarshal([]byte(value), &asset); err != nil {
		return nil, false, err
	}
	return &asset, true, nil
}

func (c userAssetCache) UpdateUserAsset(ctx context.Context, userId string, asset *model.UserAsset) error {
	var contentEncode []byte
	var err error
	if asset != nil {
		contentEncode, err = json.Marshal(asset)
		if err != nil {
			return err
		}
	} else {
		contentEncode = []byte(constant.RedisValueEmpty)
	}

	return c.redis.Set(ctx, KeyUserAsset(userId), contentEncode, TimeUserAsset()).Err()
}

func (c userAssetCache) DeleteUserAsset(ctx context.Context, userId string) (bool, error) {
	res, err := c.redis.Del(ctx, KeyUserAsset(userId)).Result()
	if err != nil {
		return false, err
	}
	if res != 1 {
		return false, nil
	}
	return true, nil
}

func (c userAssetCache) LockUserAsset(ctx context.Context, userId string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserAssetLock(userId),
		token,
		redis.SetArgs{
			Mode: "NX",
			TTL:  5 * time.Second,
		},
	).Result()
	if err != nil {
		return "", false, err
	}
	if result != "OK" {
		return "", false, nil
	}
	return token, true, nil
}

func (c userAssetCache) UnlockUserAsset(ctx context.Context, userId string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserAssetLock(userId)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_asset lock fail: error token: %s", token)
	}
	return nil
}

func (c userAssetCache) UpdateGold(ctx context.Context, userId string, gold int) error {
	return c.redis.Set(ctx, KeyUserAssetGold(userId), gold, time.Hour).Err()
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
