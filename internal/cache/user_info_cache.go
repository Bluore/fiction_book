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
	keyUserInfo     = "GSF:user:cache:user_info:%s"
	keyUserInfoLock = "GSF:user:lock:user_info:%s"

	timeUserInfo = time.Hour
)

func KeyUserInfo(userID string) string {
	return fmt.Sprintf(keyUserInfo, userID)
}

func KeyUserInfoLock(userID string) string {
	return fmt.Sprintf(keyUserInfoLock, userID)
}

func TimeUserInfo() time.Duration {
	return timeUserInfo + TimeShakeDuration(timeUserInfo)
}

type UserInfoCache interface {
	GetUserInfo(ctx context.Context, userID string) (*model.UserInfo, bool, error)
	UpdateUserInfo(ctx context.Context, userID string, userInfo *model.UserInfo) error
	DeleteUserInfo(ctx context.Context, userID string) (bool, error)

	LockUserInfo(ctx context.Context, userID string) (string, bool, error)
	UnlockUserInfo(ctx context.Context, userID string, token string) error
}

type userInfoCache struct {
	redis *redis.Client
}

func NewUserInfoCache(redis *redis.Client) UserInfoCache {
	return &userInfoCache{redis: redis}
}

func (c *userInfoCache) GetUserInfo(ctx context.Context, userID string) (*model.UserInfo, bool, error) {
	value, err := c.redis.Get(ctx, KeyUserInfo(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if value == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var userInfo model.UserInfo
	if err := json.Unmarshal([]byte(value), &userInfo); err != nil {
		return nil, false, err
	}
	return &userInfo, true, nil
}

func (c *userInfoCache) UpdateUserInfo(ctx context.Context, userID string, userInfo *model.UserInfo) error {
	var contentEncode []byte
	var err error
	if userInfo != nil {
		contentEncode, err = json.Marshal(userInfo)
		if err != nil {
			return err
		}
	} else {
		contentEncode = []byte(constant.RedisValueEmpty)
	}

	return c.redis.Set(ctx, KeyUserInfo(userID), contentEncode, TimeUserInfo()).Err()
}

func (c *userInfoCache) LockUserInfo(ctx context.Context, userID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserInfoLock(userID),
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

func (c *userInfoCache) UnlockUserInfo(ctx context.Context, userID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserInfoLock(userID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_info lock fail: error token: %s", token)
	}
	return nil
}

func (c *userInfoCache) DeleteUserInfo(ctx context.Context, userID string) (bool, error) {
	res, err := c.redis.Del(ctx, KeyUserInfo(userID)).Result()
	if err != nil {
		return false, err
	}
	if res != 1 {
		return false, nil
	}
	return true, nil
}
