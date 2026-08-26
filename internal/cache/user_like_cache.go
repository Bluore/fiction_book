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
	keyUserLike               = "GSF:user:cache:like:user_id:%s:liker_id:%s"
	keyUserLikeLock           = "GSF:user:lock:like:user_id:%s:liker_id:%s"
	keyUserLikeUserIDList     = "GSF:user:cache:like_list:liker:%s"
	keyUserLikeUserIDListLock = "GSF:user:lock:like_list:liker:%s"

	keyUserFollowerIDList     = "GSF:user:cache:follower_list:user:%s"
	keyUserFollowerIDListLock = "GSF:user:lock:follower_list:user:%s"

	timeUserLike = time.Hour
)

func KeyUserLike(userID string, likerID string) string {
	return fmt.Sprintf(keyUserLike, userID, likerID)
}

func KeyUserLikeLock(userID string, likerID string) string {
	return fmt.Sprintf(keyUserLikeLock, userID, likerID)
}

func KeyUserLikeUserIDList(likerID string) string {
	return fmt.Sprintf(keyUserLikeUserIDList, likerID)
}

func KeyUserLikeUserIDListLock(likerID string) string {
	return fmt.Sprintf(keyUserLikeUserIDListLock, likerID)
}

func KeyUserFollowerIDList(userID string) string {
	return fmt.Sprintf(keyUserFollowerIDList, userID)
}

func KeyUserFollowerIDListLock(userID string) string {
	return fmt.Sprintf(keyUserFollowerIDListLock, userID)
}

func TimeUserLike() time.Duration {
	return timeUserLike + TimeShakeDuration(timeUserLike)
}

type UserLikeCache interface {
	GetUserLike(ctx context.Context, userID string, likerID string) (*model.UserLike, bool, error)
	GetUserLikeUserIDList(ctx context.Context, likerID string) ([]string, bool, error)
	GetUserFollowerIDList(ctx context.Context, userID string) ([]string, bool, error)

	UpdateUserLike(ctx context.Context, userID string, likerID string, userLike *model.UserLike) error
	UpdateUserLikeUserIDList(ctx context.Context, likerID string, userID []string) error
	UpdateUserFollowerIDList(ctx context.Context, userID string, followerID []string) error

	DeleteUserLike(ctx context.Context, userID string, likerID string) (bool, error)
	DeleteUserLikeUserIDList(ctx context.Context, likerID string) error
	DeleteUserFollowerIDList(ctx context.Context, userID string) error

	LockUserLike(ctx context.Context, userID string, likerID string) (string, bool, error)
	LockUserLikeUserIDList(ctx context.Context, likerID string) (string, bool, error)
	LockUserFollowerIDList(ctx context.Context, userID string) (string, bool, error)
	UnlockUserLike(ctx context.Context, userID string, likerID string, token string) error
	UnlockUserLikeUserIDList(ctx context.Context, likerID string, token string) error
	UnlockUserFollowerIDList(ctx context.Context, userID string, token string) error
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

func (c userLikeCache) LockUserLike(ctx context.Context, userID string, likerID string) (string, bool, error) {
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
	res, err := c.redis.Del(ctx, KeyUserLike(userID, likerID)).Result()
	if err != nil {
		return false, err
	}
	if res != 1 {
		return false, nil
	}
	return true, nil
}

func (c userLikeCache) GetUserLikeUserIDList(ctx context.Context, likerID string) ([]string, bool, error) {
	key := KeyUserLikeUserIDList(likerID)
	exist, err := c.redis.Exists(ctx, key).Result()
	if err != nil {
		return make([]string, 0), false, err
	}
	if exist == 0 {
		return make([]string, 0), false, nil
	}

	userIDs, err := c.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return make([]string, 0), false, err
	}
	return userIDs, true, nil
}

func (c userLikeCache) GetUserFollowerIDList(ctx context.Context, userID string) ([]string, bool, error) {
	key := KeyUserFollowerIDList(userID)
	exist, err := c.redis.Exists(ctx, key).Result()
	if err != nil {
		return make([]string, 0), false, err
	}
	if exist == 0 {
		return make([]string, 0), false, nil
	}

	followerIDs, err := c.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return make([]string, 0), false, err
	}
	return followerIDs, true, nil
}

func (c userLikeCache) UpdateUserLikeUserIDList(ctx context.Context, likerID string, userID []string) error {
	key := KeyUserLikeUserIDList(likerID)
	// 先删除再整体写入，保证列表与传入内容一致
	_, err := c.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, key)
		if len(userID) > 0 {
			pipe.RPush(ctx, key, userID)
		}
		pipe.Expire(ctx, key, TimeUserLike())
		return nil
	})
	return err
}

func (c userLikeCache) UpdateUserFollowerIDList(ctx context.Context, userID string, followerID []string) error {
	key := KeyUserFollowerIDList(userID)
	// 先删除再整体写入，保证列表与传入内容一致
	_, err := c.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, key)
		if len(followerID) > 0 {
			pipe.RPush(ctx, key, followerID)
		}
		pipe.Expire(ctx, key, TimeUserLike())
		return nil
	})
	return err
}

func (c userLikeCache) DeleteUserLikeUserIDList(ctx context.Context, likerID string) error {
	return c.redis.Del(ctx, KeyUserLikeUserIDList(likerID)).Err()
}

func (c userLikeCache) DeleteUserFollowerIDList(ctx context.Context, userID string) error {
	return c.redis.Del(ctx, KeyUserFollowerIDList(userID)).Err()
}

func (c userLikeCache) LockUserLikeUserIDList(ctx context.Context, likerID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserLikeUserIDListLock(likerID),
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

func (c userLikeCache) LockUserFollowerIDList(ctx context.Context, userID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserFollowerIDListLock(userID),
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

func (c userLikeCache) UnlockUserLikeUserIDList(ctx context.Context, likerID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserLikeUserIDListLock(likerID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_like_user_id_list lock fail: error token: %s", token)
	}
	return nil
}

func (c userLikeCache) UnlockUserFollowerIDList(ctx context.Context, userID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserFollowerIDListLock(userID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_follower_id_list lock fail: error token: %s", token)
	}
	return nil
}
