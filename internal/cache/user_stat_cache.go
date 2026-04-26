package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	keyUserStatReadBookNum  = "Data:User:Stat:%s:ReadBookNum"
	keyUserStatReadWordsNum = "Data:User:Stat:%s:ReadWordsNum"
)

func KeyUserStatReadBookNum(userId string) string {
	return fmt.Sprintf(keyUserStatReadBookNum, userId)
}

func KeyUserStatReadWords(userId string) string {
	return fmt.Sprintf(keyUserStatReadWordsNum, userId)
}

type UserStatCache interface {
	IncreaseReadBookNum(ctx context.Context, userId string, increaseNum int64) error
	GetReadBookNum(ctx context.Context, userId string) (int64, bool, error)
	IncreaseReadWordsNum(ctx context.Context, userId string, increaseNum int64) error
	GetReadWordsNum(ctx context.Context, userId string) (int64, bool, error)
}

type userStatCache struct {
	redis *redis.Client
}

func NewUserStatCache(redis *redis.Client) UserStatCache {
	return &userStatCache{redis: redis}
}

func (c userStatCache) IncreaseReadBookNum(ctx context.Context, userId string, increaseNum int64) error {
	return c.redis.IncrBy(ctx, KeyUserStatReadBookNum(userId), increaseNum).Err()
}

func (c userStatCache) GetReadBookNum(ctx context.Context, userId string) (int64, bool, error) {
	cache, err := c.redis.Get(ctx, KeyUserStatReadBookNum(userId)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	result, err := strconv.ParseInt(cache, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return result, true, nil
}

func (c userStatCache) IncreaseReadWordsNum(ctx context.Context, userId string, increaseNum int64) error {
	return c.redis.IncrBy(ctx, KeyUserStatReadWords(userId), increaseNum).Err()
}

func (c userStatCache) GetReadWordsNum(ctx context.Context, userId string) (int64, bool, error) {
	cache, err := c.redis.Get(ctx, KeyUserStatReadWords(userId)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	result, err := strconv.ParseInt(cache, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return result, true, nil
}
