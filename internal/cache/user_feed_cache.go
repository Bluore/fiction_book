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
	keyFeedDetail      = "GSF:feed:cache:feed:%s"
	keyUserFeedCursor  = "GSF:feed:cache:user:%s:cursor:%s"
	keyUserFeedFeedIDs = "GSF:feed:cache:user:%s:feed_ids:%s"

	keyFeedDetailLock      = "GSF:feed:lock:feed:%s"
	keyUserFeedCursorLock  = "GSF:feed:lock:user:%s:cursor:%s"
	keyUserFeedFeedIDsLock = "GSF:feed:lock:user:%s:feed_ids:%s"

	timeUserFeed = time.Hour
)

func KeyFeedDetail(feedID string) string {
	return fmt.Sprintf(keyFeedDetail, feedID)
}

func KeyUserFeedCursor(userID string, cursorID string) string {
	return fmt.Sprintf(keyUserFeedCursor, userID, cursorID)
}

func KeyUserFeedFeedIDs(userID string, cursorID string) string {
	return fmt.Sprintf(keyUserFeedFeedIDs, userID, cursorID)
}

func KeyFeedDetailLock(feedID string) string {
	return fmt.Sprintf(keyFeedDetailLock, feedID)
}

func KeyUserFeedCursorLock(userID string, cursorID string) string {
	return fmt.Sprintf(keyUserFeedCursorLock, userID, cursorID)
}

func KeyUserFeedFeedIDsLock(userID string, cursorID string) string {
	return fmt.Sprintf(keyUserFeedFeedIDsLock, userID, cursorID)
}

func TimeUserFeed() time.Duration {
	return timeUserFeed + TimeShakeDuration(timeUserFeed)
}

type UserFeedCache interface {
	// GetFeedDetail 获取Feed具体内容
	GetFeedDetail(ctx context.Context, feedID string) (feedDetail *model.UserFeed, feedDetailHit bool, err error)
	// GetUserFeedCursor 获取Cursor信息
	GetUserFeedCursor(ctx context.Context, userID string, cursorID string) (cursorDetail *model.UserFeedCursor, cursorDetailHit bool, err error)
	// GetUserFeedFeedIDs 获取FeedID列表，带Limit
	GetUserFeedFeedIDs(ctx context.Context, userID string, cursorID string, offset int, limit int) (feedIDs []string, feedIDsHit bool, err error)

	// UpdateFeedDetail 获取Feed具体内容
	UpdateFeedDetail(ctx context.Context, feedID string, feedDetail *model.UserFeed) (err error)
	// UpdateUserFeedCursor 更新Cursor信息
	UpdateUserFeedCursor(ctx context.Context, userID string, cursorID string, cursorDetail *model.UserFeedCursor) (err error)
	// UpdateUserFeedFeedIDs 更新FeedID列表信息
	UpdateUserFeedFeedIDs(ctx context.Context, userID string, cursorID string, feedIDs []string) (err error)

	// DeleteFeedDetail 删除feed信息
	DeleteFeedDetail(ctx context.Context, feedID string) error
	// DeleteUserFeedCursor 删除Cursor信息
	DeleteUserFeedCursor(ctx context.Context, userID string, cursorID string) error
	// DeleteUserFeedFeedIDs 删除FeedID列表
	DeleteUserFeedFeedIDs(ctx context.Context, userID string, cursorID string) error

	// Deprecated: LockFeedDetail 锁住Feed
	LockFeedDetail(ctx context.Context, feedID string) (token string, lockSuccess bool, err error)
	// Deprecated: LockUserFeedCursor 锁住Cursor
	LockUserFeedCursor(ctx context.Context, userID string, cursorID string) (token string, lockSuccess bool, err error)
	// Deprecated: LockUserFeedFeedIDs 锁住FeedID列表
	LockUserFeedFeedIDs(ctx context.Context, userID string, cursorID string) (token string, lockSuccess bool, err error)

	// Deprecated: UnlockFeedDetail 释放锁Feed
	UnlockFeedDetail(ctx context.Context, feedID string, token string) error
	// Deprecated: UnlockUserFeedCursor 释放锁Cursor
	UnlockUserFeedCursor(ctx context.Context, userID string, cursorID string, token string) error
	// Deprecated: UnlockUserFeedFeedIDs 释放锁FeedID列表
	UnlockUserFeedFeedIDs(ctx context.Context, userID string, cursorID string, token string) error
}

type userFeedCache struct {
	redis *redis.Client
}

func NewUserFeedCache(redis *redis.Client) UserFeedCache {
	return &userFeedCache{
		redis: redis,
	}
}

// GetFeedDetail 获取Feed具体内容
func (c *userFeedCache) GetFeedDetail(ctx context.Context, feedID string) (*model.UserFeed, bool, error) {
	value, err := c.redis.Get(ctx, KeyFeedDetail(feedID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if value == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var feedDetail model.UserFeed
	if err := json.Unmarshal([]byte(value), &feedDetail); err != nil {
		return nil, false, err
	}
	return &feedDetail, true, nil
}

// UpdateFeed 更新Feed具体内容
func (c *userFeedCache) UpdateFeedDetail(ctx context.Context, feedID string, feedDetail *model.UserFeed) error {
	var feedEncode []byte
	var err error
	if feedDetail != nil {
		feedEncode, err = json.Marshal(feedDetail)
		if err != nil {
			return err
		}
	} else {
		feedEncode = []byte(constant.RedisValueEmpty)
	}

	return c.redis.Set(ctx, KeyFeedDetail(feedID), feedEncode, TimeUserFeed()).Err()
}

// DeleteFeedDetail 删除feed信息
func (c *userFeedCache) DeleteFeedDetail(ctx context.Context, feedID string) error {
	return c.redis.Del(ctx, KeyFeedDetail(feedID)).Err()
}

// GetUserFeedCursor 获取Cursor信息
func (c *userFeedCache) GetUserFeedCursor(ctx context.Context, userID string, cursorID string) (*model.UserFeedCursor, bool, error) {
	value, err := c.redis.Get(ctx, KeyUserFeedCursor(userID, cursorID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if value == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var cursorDetail model.UserFeedCursor
	if err := json.Unmarshal([]byte(value), &cursorDetail); err != nil {
		return nil, false, err
	}
	return &cursorDetail, true, nil
}

// UpdateUserFeedCursor 更新Cursor信息
func (c *userFeedCache) UpdateUserFeedCursor(ctx context.Context, userID string, cursorID string, cursorDetail *model.UserFeedCursor) error {
	var cursorEncode []byte
	var err error
	if cursorDetail != nil {
		cursorEncode, err = json.Marshal(cursorDetail)
		if err != nil {
			return err
		}
	} else {
		cursorEncode = []byte(constant.RedisValueEmpty)
	}

	return c.redis.Set(ctx, KeyUserFeedCursor(userID, cursorID), cursorEncode, TimeUserFeed()).Err()
}

// DeleteUserFeedCursor 删除Cursor信息
func (c *userFeedCache) DeleteUserFeedCursor(ctx context.Context, userID string, cursorID string) error {
	return c.redis.Del(ctx, KeyUserFeedCursor(userID, cursorID)).Err()
}

// GetUserFeedFeedIDs 获取FeedID列表，带Limit
func (c *userFeedCache) GetUserFeedFeedIDs(ctx context.Context, userID string, cursorID string, offset int, limit int) ([]string, bool, error) {
	key := KeyUserFeedFeedIDs(userID, cursorID)
	exist, err := c.redis.Exists(ctx, key).Result()
	if err != nil {
		return nil, false, err
	}
	if exist == 0 {
		return nil, false, nil
	}
	if limit <= 0 {
		return make([]string, 0), true, nil
	}

	feedIDs, err := c.redis.LRange(ctx, key, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, false, err
	}
	return feedIDs, true, nil
}

// UpdateUserFeedFeedIDs 更新FeedID列表信息
func (c *userFeedCache) UpdateUserFeedFeedIDs(ctx context.Context, userID string, cursorID string, feedIDs []string) error {
	key := KeyUserFeedFeedIDs(userID, cursorID)
	// 先删除再整体写入，保证列表与传入内容一致
	_, err := c.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, key)
		if len(feedIDs) > 0 {
			pipe.RPush(ctx, key, feedIDs)
		}
		pipe.Expire(ctx, key, TimeUserFeed())
		return nil
	})
	return err
}

// DeleteUserFeedFeedIDs 删除FeedID列表
func (c *userFeedCache) DeleteUserFeedFeedIDs(ctx context.Context, userID string, cursorID string) error {
	return c.redis.Del(ctx, KeyUserFeedFeedIDs(userID, cursorID)).Err()
}

// LockFeedDetail 锁住Feed
func (c *userFeedCache) LockFeedDetail(ctx context.Context, feedID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyFeedDetailLock(feedID),
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

// LockUserFeedCursor 锁住Cursor
func (c *userFeedCache) LockUserFeedCursor(ctx context.Context, userID string, cursorID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserFeedCursorLock(userID, cursorID),
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

// LockUserFeedFeedIDs 锁住FeedID列表
func (c *userFeedCache) LockUserFeedFeedIDs(ctx context.Context, userID string, cursorID string) (string, bool, error) {
	token := uuid.NewString()
	result, err := c.redis.SetArgs(
		ctx,
		KeyUserFeedFeedIDsLock(userID, cursorID),
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

// UnlockFeedDetail 释放锁Feed
func (c *userFeedCache) UnlockFeedDetail(ctx context.Context, feedID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyFeedDetailLock(feedID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock feed_detail lock fail: error token: %s", token)
	}
	return nil
}

// UnlockUserFeedCursor 释放锁Cursor
func (c *userFeedCache) UnlockUserFeedCursor(ctx context.Context, userID string, cursorID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserFeedCursorLock(userID, cursorID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_feed_cursor lock fail: error token: %s", token)
	}
	return nil
}

// UnlockUserFeedFeedIDs 释放锁FeedID列表
func (c *userFeedCache) UnlockUserFeedFeedIDs(ctx context.Context, userID string, cursorID string, token string) error {
	unlockScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return 0
`)
	result, err := unlockScript.Run(
		ctx,
		c.redis,
		[]string{KeyUserFeedFeedIDsLock(userID, cursorID)},
		token,
	).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("unlock user_feed_feed_ids lock fail: error token: %s", token)
	}
	return nil
}
