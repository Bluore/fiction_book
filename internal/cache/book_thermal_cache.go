package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyBookThermalValue    = "GSF:book:cache:book:%s:thermal"
	keyBookThermalTimeLock = "GSF:book:lock:book:%s:thermal"

	bookThermalValueCacheTime = time.Duration(5 * time.Minute)
	bookThermalLockCacheTime  = time.Duration(1 * time.Minute)
)

var (
	// ErrorBookThermalLockExist 热度更新锁已存在，说明该书籍在锁有效期内已经更新过
	ErrorBookThermalLockExist = errors.New("ErrorBookThermalLockExist")
	// ErrorBookThermalValueNotExist 热度值缓存不存在，需先从数据库加载后用 UpdateBookThermalValue 创建
	ErrorBookThermalValueNotExist = errors.New("ErrorBookThermalValueNotExist")
)

type BookThermalValue = uint64

type BookThermalCache interface {
	// GetBookThermalValue Get BookThermalValue value
	GetBookThermalValue(ctx context.Context, bookId string) (value BookThermalValue, exist bool, err error)
	// UpdateBookThermalValue Create BookThermalValue Key, if key is nil
	UpdateBookThermalValue(ctx context.Context, bookId string, value BookThermalValue) (err error)
	// AddBookThermalValue Add BookThermalValue value, if key is not nil
	AddBookThermalValue(ctx context.Context, bookId string, value BookThermalValue) (err error)

	// LockBookThermalValue Create BookThermalTimeLock key
	LockBookThermalValue(ctx context.Context, bookId string) (err error)
}

type bookThermalCache struct {
	redis *redis.Client
}

func NewBookThermalCache(redis *redis.Client) BookThermalCache {
	return &bookThermalCache{redis: redis}
}

func (c *bookThermalCache) GetBookThermalValue(ctx context.Context, bookId string) (BookThermalValue, bool, error) {
	value, err := c.redis.Get(ctx, KeyBookThermalValue(bookId)).Int64()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return BookThermalValue(value), true, nil
}

func (c *bookThermalCache) UpdateBookThermalValue(ctx context.Context, bookId string, value BookThermalValue) error {
	// NX 模式只在 key 不存在时才写入，已存在时不覆盖已有热度值
	_, err := c.redis.SetArgs(ctx, KeyBookThermalValue(bookId), int64(value), redis.SetArgs{
		Mode: "NX",
		TTL:  bookThermalValueCacheTime,
	}).Result()
	// key 已存在时 redis 返回 nil 回复，go-redis 转换为 redis.Nil，这里视为无需写入
	if err == redis.Nil {
		return nil
	}
	return err
}

func (c *bookThermalCache) AddBookThermalValue(ctx context.Context, bookId string, value BookThermalValue) error {
	key := KeyBookThermalValue(bookId)
	if err := c.redis.Get(ctx, key).Err(); err != nil {
		if err == redis.Nil {
			return ErrorBookThermalValueNotExist
		}
		return err
	}

	// 热度值自增，key 不存在时 redis 会先初始化为 0 再自增
	if err := c.redis.IncrBy(ctx, key, int64(value)).Err(); err != nil {
		return err
	}

	// 如果没有过期时间则设置过期时间
	ttlRes := c.redis.TTL(ctx, key)
	if ttlRes.Err() != nil {
		return ttlRes.Err()
	}
	if ttlRes.Val() == -1 {
		return c.redis.Expire(ctx, key, bookThermalValueCacheTime).Err()
	}
	return nil
}

func (c *bookThermalCache) LockBookThermalValue(ctx context.Context, bookId string) error {
	// NX 模式只有在 key 不存在时才能加锁成功，锁到期自动释放，无需手动解锁
	_, err := c.redis.SetArgs(ctx, BookThermalTimeLockKey(bookId), "true", redis.SetArgs{
		Mode: "NX",
		TTL:  bookThermalLockCacheTime,
	}).Result()
	// 锁已存在时 redis 返回 nil 回复，go-redis 转换为 redis.Nil
	if err == redis.Nil {
		return ErrorBookThermalLockExist
	}
	return err
}

// KeyBookThermalValue 书籍热度值缓存 key
func KeyBookThermalValue(bookId string) string {
	return fmt.Sprintf(keyBookThermalValue, bookId)
}

// BookThermalTimeLockKey 书籍热度更新锁 key
func BookThermalTimeLockKey(bookId string) string {
	return fmt.Sprintf(keyBookThermalTimeLock, bookId)
}
