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
	keyUserBookshelf     = "GSF:user:cache:bookshelf:%s:%s"
	keyUserBookshelfLock = "GSF:user:lock:bookshelf:%s:%s"

	timeUserBookshelf = time.Hour
)

func KeyUserBookshelf(userID string, bookID string) string {
	return fmt.Sprintf(keyUserBookshelf, userID, bookID)
}

func KeyUserBookshelfLock(userID string, bookID string) string {
	return fmt.Sprintf(keyUserBookshelfLock, userID, bookID)
}

func TimeUserBookshelf() time.Duration {
	return timeUserBookshelf + TimeShakeDuration(timeUserBookshelf)
}

type UserBookshelfCache interface {
	GetUserBookshelf(ctx context.Context, userID string, bookID string) (*model.UserBookshelf, bool, error)

	UpdateUserBookshelf(ctx context.Context, userID string, bookID string, userBookshelf *model.UserBookshelf) error

	DeleteUserBookshelf(ctx context.Context, userID string, bookID string) (bool, error)

	LockUserBookshelf(ctx context.Context, userID string, bookID string) (string, bool, error)
	UnlockUserBookshelf(ctx context.Context, userID string, bookID string, token string) error
}

type userBookshelfCache struct {
	redis *redis.Client
}

func NewUserBookshelfCache(redis *redis.Client) UserBookshelfCache {
	return &userBookshelfCache{
		redis: redis,
	}
}

func (c userBookshelfCache) GetUserBookshelf(ctx context.Context, userID string, bookID string) (*model.UserBookshelf, bool, error) {
	value, err := c.redis.Get(ctx, KeyUserBookshelf(userID, bookID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if value == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var userBookshelf model.UserBookshelf
	if err := json.Unmarshal([]byte(value), &userBookshelf); err != nil {
		return nil, false, err
	}
	return &userBookshelf, true, nil
}

func (c userBookshelfCache) UpdateUserBookshelf(ctx context.Context, userID string, bookID string, userBookshelf *model.UserBookshelf) error {
	var contentEncode []byte
	var err error
	if userBookshelf != nil {
		contentEncode, err = json.Marshal(userBookshelf)
		if err != nil {
			return err
		}
	} else {
		contentEncode = []byte(constant.RedisValueEmpty)
	}

	return c.redis.Set(ctx, KeyUserBookshelf(userID, bookID), contentEncode, TimeUserBookshelf()).Err()
}

func (c userBookshelfCache) LockUserBookshelf(ctx context.Context, userID string, bookID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserBookshelfLock(userID, bookID),
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

func (c userBookshelfCache) UnlockUserBookshelf(ctx context.Context, userID string, bookID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserBookshelfLock(userID, bookID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_bookshelf lock fail: error token: %s", token)
	}
	return nil
}

func (c userBookshelfCache) DeleteUserBookshelf(ctx context.Context, userID string, bookID string) (bool, error) {
	res, err := c.redis.Del(ctx, KeyUserBookshelf(userID, bookID)).Result()
	if err != nil {
		return false, err
	}
	if res != 1 {
		return false, nil
	}
	return true, nil
}
