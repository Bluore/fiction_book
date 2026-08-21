package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fiction-book/internal/model"
	"fiction-book/pkg/constant"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	keyUser     = "GSF:user:cache:user:%s"
	keyUserLock = "GSF:user:lock:user:%s"

	timeUser = time.Hour
)

func KeyUser(userID string) string {
	return fmt.Sprintf(keyUser, userID)
}

func KeyUserLock(userID string) string {
	return fmt.Sprintf(keyUserLock, userID)
}

func TimeUser() time.Duration {
	return timeUser + TimeShakeDuration(timeUser)
}

type UserCache interface {
	GetUser(ctx context.Context, userID string) (*model.User, bool, error)
	UpdateUser(ctx context.Context, userID string, user *model.User) error
	DeleteUser(ctx context.Context, userID string) (bool, error)

	LockUser(ctx context.Context, userID string) (string, bool, error)
	UnlockUser(ctx context.Context, userID string, token string) error
}

type userCache struct {
	redis *redis.Client
}

func NewUserCache(redis *redis.Client) UserCache {
	return &userCache{
		redis: redis,
	}
}

func (c *userCache) GetUser(ctx context.Context, userID string) (*model.User, bool, error) {
	value, err := c.redis.Get(ctx, KeyUser(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if value == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var user model.User
	if err := json.Unmarshal([]byte(value), &user); err != nil {
		return nil, false, err
	}
	return &user, true, nil
}

func (c *userCache) UpdateUser(ctx context.Context, userID string, user *model.User) error {
	var contentEncode []byte
	var err error
	if user != nil {
		contentEncode, err = json.Marshal(user)
		if err != nil {
			return err
		}
	} else {
		contentEncode = []byte(constant.RedisValueEmpty)
	}

	return c.redis.Set(ctx, KeyUser(userID), contentEncode, TimeUser()).Err()
}

func (c *userCache) LockUser(ctx context.Context, userID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserLock(userID),
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

func (c *userCache) UnlockUser(ctx context.Context, userID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserLock(userID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user lock fail: error token: %s", token)
	}
	return nil
}

func (c *userCache) DeleteUser(ctx context.Context, userID string) (bool, error) {
	res, err := c.redis.Del(ctx, KeyUser(userID)).Result()
	if err != nil {
		return false, err
	}
	if res != 1 {
		return false, nil
	}
	return true, nil
}
