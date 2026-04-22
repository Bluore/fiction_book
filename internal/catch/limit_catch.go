package catch

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyIsUserBrowse5Min     = "Added:User:%s:Browse:5Min"
	keyIsUserBrowseBook5Min = "Added:User:%s:Browse:%s:5Min"
)

type LimitCatch interface {
	CheckUserBrowse5Min(ctx context.Context, userId string) (bool, error)
	AddUserBrowse5Min(ctx context.Context, userId string) error
	CheckUserBrowseBook5Min(ctx context.Context, userId string, bookId string) (bool, error)
	AddUserBrowseBook5Min(ctx context.Context, userId string, bookId string) error
}
type limitCatch struct {
	redis *redis.Client
}

func NewLimitCatch(redis *redis.Client) LimitCatch {
	return &limitCatch{redis: redis}
}

func KeyIsUserBrowse5Min(userId string) string {
	return fmt.Sprintf(keyIsUserBrowse5Min, userId)
}

func KeyIsUserBrowseBook5Min(userId string, bookId string) string {
	return fmt.Sprintf(keyIsUserBrowseBook5Min, userId, bookId)
}

func (c limitCatch) CheckUserBrowse5Min(ctx context.Context, userId string) (bool, error) {
	_, err := c.redis.Exists(ctx, KeyIsUserBrowse5Min(userId)).Result()
	if err != nil {
		return false, err
	}
	return true, err
}

func (c limitCatch) AddUserBrowse5Min(ctx context.Context, userId string) error {
	return c.redis.Set(ctx, KeyIsUserBrowse5Min(userId), 1, 5*time.Minute).Err()
}

func (c limitCatch) CheckUserBrowseBook5Min(ctx context.Context, userId string, bookId string) (bool, error) {
	_, err := c.redis.Exists(ctx, KeyIsUserBrowseBook5Min(userId, bookId)).Result()
	if err != nil {
		return false, err
	}
	return true, err
}

func (c limitCatch) AddUserBrowseBook5Min(ctx context.Context, userId string, bookId string) error {
	return c.redis.Set(ctx, KeyIsUserBrowseBook5Min(userId, bookId), 1, 5*time.Minute).Err()
}
