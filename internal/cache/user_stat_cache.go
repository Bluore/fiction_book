package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	keyUserStatReadBookNum     = "Data:User:%s:Stat:ReadBook:Num"
	keyUserStatReadBookList    = "Data:User:%s:Stat:ReadBook:List"
	keyUserStatReadBookListBAK = "Data:User:%s:Stat:ReadBook:BAK:List"
	keyUserStatReadBookHash    = "Data:User:%s:Stat:ReadBook:Hash"
	keyUserStatReadWordsNum    = "Data:User:%s:Stat:ReadWordsNum"
)

func KeyUserStatReadBookNum(userId string) string {
	return fmt.Sprintf(keyUserStatReadBookNum, userId)
}

func KeyUserStatReadWords(userId string) string {
	return fmt.Sprintf(keyUserStatReadWordsNum, userId)
}

func KeyUserStatReadBookList(userId string) string {
	return fmt.Sprintf(keyUserStatReadBookList, userId)
}

func KeyUserStatReadBookHash(userId string) string {
	return fmt.Sprintf(keyUserStatReadBookHash, userId)
}

func KeyUserStatReadBookListBAK(userId string) string {
	return fmt.Sprintf(keyUserStatReadBookListBAK, userId)
}

type UserStatCache interface {
	AddReadBook(ctx context.Context, userId string, bookId string) (bool, error)
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

func (c userStatCache) AddReadBook(ctx context.Context, userId string, bookId string) (bool, error) {
	count, err := c.redis.HLen(ctx, KeyUserStatReadBookHash(userId)).Result()
	if err != nil {
		return false, err
	}
	// 如果记录数量>1000个，只保留最近500条
	if count > 1000 {
		// 备份一下数据
		listRes, err := c.redis.LRange(ctx, KeyUserStatReadBookList(userId), 0, 499).Result()
		if err != nil {
			return false, err
		}
		c.redis.Del(ctx, KeyUserStatReadBookListBAK(userId))
		err = c.redis.LPush(ctx, KeyUserStatReadBookList(userId), listRes).Err()
		if err != nil {
			return false, err
		}
		// 删除并保留部分数据
		err = func() error {
			err = c.redis.Del(ctx, KeyUserStatReadBookList(userId)).Err()
			if err != nil {
				return err
			}

			err = c.redis.LPush(ctx, KeyUserStatReadBookList(userId), listRes).Err()
			if err != nil {
				return err
			}

			err = c.redis.Del(ctx, KeyUserStatReadBookHash(userId)).Err()
			if err != nil {
				return err
			}

			var hashInsert []string
			for _, h := range listRes {
				hashInsert = append(hashInsert, h)
				hashInsert = append(hashInsert, "1")
			}

			err = c.redis.HSet(ctx, KeyUserStatReadBookNum(userId), hashInsert).Err()
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			// 尝试恢复备份数据
			bakErr := c.redis.Del(ctx, KeyUserStatReadBookList(userId)).Err()
			if bakErr != nil {
				return false, bakErr
			}

			bakErr = c.redis.LPush(ctx, KeyUserStatReadBookList(userId), listRes).Err()
			if bakErr != nil {
				return false, bakErr
			}

			return false, err
		}
	}

	setRes, err := c.redis.HSet(ctx, KeyUserStatReadBookHash(userId), bookId, 1).Result()
	if err != nil {
		return false, err
	}
	if setRes == 0 {
		return false, nil
	}
	err = c.redis.LPush(ctx, KeyUserStatReadBookList(userId), bookId).Err()
	if err != nil {
		return false, err
	}

	err = c.redis.IncrBy(ctx, KeyUserStatReadBookNum(userId), 1).Err()
	if err != nil {
		return false, err
	}
	return true, nil
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
