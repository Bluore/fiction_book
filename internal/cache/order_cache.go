package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/model"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	keyOrder          = "Cache:Order:%s"
	keyOrderGoroutine = "Lock:Order:Goroutine:%s"

	ErrorOrderCreateGoroutineExist = errors.New("ErrorOrderCreateGoroutineExist")
)

func KeyOrder(orderId string) string {
	return fmt.Sprintf(keyOrder, orderId)
}

func OrderGoroutineLockKey(orderId string) string {
	return fmt.Sprintf(keyOrderGoroutine, orderId)
}

type OrderCache interface {
	Set(ctx context.Context, orderId string, order *model.Order) error
	Get(ctx context.Context, orderId string) (*model.Order, bool, error)
	Delete(ctx context.Context, orderId string) error

	// 创建Goroutine标识符
	LockGoroutine(ctx context.Context, orderId string, deadTime time.Duration) error
	UnlockGoroutine(ctx context.Context, orderId string) error
}

type orderCache struct {
	redis *redis.Client
}

func NewOrderCache(redis *redis.Client) OrderCache {
	return &orderCache{redis: redis}
}

func (c orderCache) Set(ctx context.Context, orderId string, order *model.Order) error {
	orderEncode, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, KeyOrder(orderId), orderEncode, time.Hour*24).Err()
}

func (c orderCache) Get(ctx context.Context, orderId string) (*model.Order, bool, error) {
	cache, err := c.redis.Get(ctx, KeyOrder(orderId)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var order model.Order
	err = json.Unmarshal([]byte(cache), &order)
	if err != nil {
		return nil, false, err
	}
	return &order, true, nil
}

func (c orderCache) Delete(ctx context.Context, orderId string) error {
	return c.redis.Del(ctx, KeyOrder(orderId)).Err()
}

func (c orderCache) LockGoroutine(ctx context.Context, orderId string, deadTime time.Duration) error {
	key := OrderGoroutineLockKey(orderId)
	// 使用 Set 配合 SetArgs 的 NX 模式实现分布式锁，只有当 key 不存在时才能设置成功
	success, err := c.redis.SetArgs(ctx, key, "true", redis.SetArgs{
		Mode: "NX",
		TTL:  deadTime,
	}).Result()
	if err != nil {
		return err
	}
	// 如果返回值为空字符串，说明 key 已存在，设置失败
	if success == "" {
		return ErrorOrderCreateGoroutineExist
	}
	return nil
}

func (c orderCache) UnlockGoroutine(ctx context.Context, orderId string) error {
	key := OrderGoroutineLockKey(orderId)
	return c.redis.Del(ctx, key).Err()
}
