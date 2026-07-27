package cache

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	keyOrder = "Cache:Order:%s"
)

func KeyOrder(orderId string) string {
	return fmt.Sprintf(keyOrder, orderId)
}

type OrderCache interface {
	Set(ctx context.Context, orderId string, order *model.Order) error
	Get(ctx context.Context, orderId string) (*model.Order, bool, error)
	Delete(ctx context.Context, orderId string) error
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
