package cache

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyBookTags = "GSF:book:cache:tags:%s:detail"
)

type BookTagCache interface {
	GetBookTags(ctx context.Context, bookID string) ([]model.BookTag, bool, error)
	SetBookTags(ctx context.Context, bookID string, tags []model.BookTag) error
	DeleteBookTags(ctx context.Context, bookID string) error
}

type bookTagCache struct {
	redis *redis.Client
}

func NewBookTagCache(redis *redis.Client) BookTagCache {
	return &bookTagCache{redis: redis}
}

func keyBookTagsFmt(bookID string) string {
	return fmt.Sprintf(keyBookTags, bookID)
}

func (c *bookTagCache) GetBookTags(ctx context.Context, bookID string) ([]model.BookTag, bool, error) {
	res, err := c.redis.Get(ctx, keyBookTagsFmt(bookID)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var tags []model.BookTag
	if err := json.Unmarshal([]byte(res), &tags); err != nil {
		return nil, false, err
	}
	return tags, true, nil
}

func (c *bookTagCache) SetBookTags(ctx context.Context, bookID string, tags []model.BookTag) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, keyBookTagsFmt(bookID), data, time.Hour).Err()
}

func (c *bookTagCache) DeleteBookTags(ctx context.Context, bookID string) error {
	return c.redis.Del(ctx, keyBookTagsFmt(bookID)).Err()
}
