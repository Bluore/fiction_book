package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyIsUserBrowse1Min         = "GSF:user:limit:browse:%s:1m"
	keyIsUserBrowseBook5Min     = "GSF:user:limit:browse_book:%s_%s:5m"
	keyIsUserBookshelfBook1Hour = "GSF:user:limit:bookshelf_book:%s_%s:1h"
)

type LimitCache interface {
	CheckUserBrowse1Min(ctx context.Context, userId string) (bool, error)
	AddUserBrowse1Min(ctx context.Context, userId string) error
	CheckUserBrowseBook5Min(ctx context.Context, userId string, bookId string) (bool, error)
	AddUserBrowseBook5Min(ctx context.Context, userId string, bookId string) error
	CheckUserBookshelfBook1Hour(ctx context.Context, userId string, bookId string) (bool, error)
	AddUserBookshelfBook1Hour(ctx context.Context, userId string, bookId string) error
}
type limitCache struct {
	redis *redis.Client
}

func NewLimitCatch(redis *redis.Client) LimitCache {
	return &limitCache{redis: redis}
}

func KeyIsUserBrowse1Min(userId string) string {
	return fmt.Sprintf(keyIsUserBrowse1Min, userId)
}

func KeyIsUserBrowseBook5Min(userId string, bookId string) string {
	return fmt.Sprintf(keyIsUserBrowseBook5Min, userId, bookId)
}

func KeyIsUserBookshelfBook1Hour(userId string, bookId string) string {
	return fmt.Sprintf(keyIsUserBookshelfBook1Hour, userId, bookId)
}

func (c limitCache) CheckUserBrowse1Min(ctx context.Context, userId string) (bool, error) {
	exist, err := c.redis.Exists(ctx, KeyIsUserBrowse1Min(userId)).Result()
	if err != nil {
		return false, err
	}
	if exist == 0 {
		return false, nil
	}
	return true, err
}

func (c limitCache) AddUserBrowse1Min(ctx context.Context, userId string) error {
	return c.redis.Set(ctx, KeyIsUserBrowse1Min(userId), 1, time.Minute).Err()
}

func (c limitCache) CheckUserBrowseBook5Min(ctx context.Context, userId string, bookId string) (bool, error) {
	exist, err := c.redis.Exists(ctx, KeyIsUserBrowseBook5Min(userId, bookId)).Result()
	if err != nil {
		return false, err
	}
	if exist == 0 {
		return false, nil
	}
	return true, err
}

func (c limitCache) AddUserBrowseBook5Min(ctx context.Context, userId string, bookId string) error {
	return c.redis.Set(ctx, KeyIsUserBrowseBook5Min(userId, bookId), 1, 5*time.Minute).Err()
}

func (c limitCache) CheckUserBookshelfBook1Hour(ctx context.Context, userId string, bookId string) (bool, error) {
	exist, err := c.redis.Exists(ctx, KeyIsUserBookshelfBook1Hour(userId, bookId)).Result()
	if err != nil {
		return false, err
	}
	if exist == 0 {
		return false, nil
	}
	return true, err
}

func (c limitCache) AddUserBookshelfBook1Hour(ctx context.Context, userId string, bookId string) error {
	return c.redis.Set(ctx, KeyIsUserBookshelfBook1Hour(userId, bookId), 1, 1*time.Hour).Err()
}
