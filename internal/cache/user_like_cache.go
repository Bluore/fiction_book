package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/model"
	"fiction-book/pkg/constant"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	keyUserLike     = "GSF:user:cache:like:%s:%s"
	keyUserLikeLock = "GSF:user:lock:like:%s:%s"

	timeUserLike = time.Hour
)

func KeyUserLike(userID string, likerID string) string {
	return fmt.Sprintf(keyUserLike, userID, likerID)
}

func KeyUserLikeLock(userID string, likerID string) string {
	return fmt.Sprintf(keyUserLikeLock, userID, likerID)
}

func TimeUserLike() time.Duration {
	return timeUserLike + TimeShakeDuration(timeUserLike)
}

type UserLikeCache interface {
	GetUserLike(ctx context.Context, userID string, likerID string) (*model.UserLike, bool, error)
	UpdateUserLike(ctx context.Context, userID string, likerID string, userLike *model.UserLike) error
	DeleteUserLike(ctx context.Context, userID string, likerID string) (bool, error)

	LockUserLike(ctx context.Context, userID string, likerID string) (string, bool, error)
	UnlockUserLike(ctx context.Context, userID string, likerID string, token string) error
}

type userLikeCache struct {
	redis *redis.Client
}

func NewUserLikeCache(redis *redis.Client) UserLikeCache {
	return &userLikeCache{
		redis: redis,
	}
}

func (c userLikeCache) GetUserLike(ctx context.Context, userID string, likerID string) (*model.UserLike, bool, error) {
	value, err := c.redis.Get(ctx, KeyUserLike(userID, likerID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if value == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var userLike model.UserLike
	if err := json.Unmarshal([]byte(value), &userLike); err != nil {
		return nil, false, err
	}
	return &userLike, true, nil
}

func (c userLikeCache) UpdateUserLike(ctx context.Context, userID string, likerID string, userLike *model.UserLike) error {
	var contentEncode []byte
	var err error
	if userLike != nil {
		contentEncode, err = json.Marshal(userLike)
		if err != nil {
			return err
		}
	} else {
		contentEncode = []byte(constant.RedisValueEmpty)
	}

	return c.redis.Set(ctx, KeyUserLike(userID, likerID), contentEncode, TimeUserLike()).Err()
}

func (c userLikeCache) LockUserLike(ctx context.Context, likerID string, userID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserLikeLock(userID, likerID),
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

func (c userLikeCache) UnlockUserLike(ctx context.Context, userID string, likerID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserLikeLock(userID, likerID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_like lock fail: error token: %s", token)
	}
	return nil
}

func (c userLikeCache) DeleteUserLike(ctx context.Context, userID string, likerID string) (bool, error) {
	res, err := c.redis.Del(ctx, KeyUserLikeLock(userID, likerID)).Result()
	if err != nil {
		return false, err
	}
	if res != 1 {
		return false, nil
	}
	return true, nil
}
