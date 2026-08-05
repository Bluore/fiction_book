package cache

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyBook = "GSF:book:cache:book:%s:detail"
)

type BookCache interface {
	GetBook(ctx context.Context, id string) (*model.Book, bool, error)
	UpdateBook(ctx context.Context, book *model.Book) error
	DeleteBook(ctx context.Context, id string) error
	GetBookThermalDaily(ctx context.Context, page int, size int) ([]string, error)
	IncreaseBookThermalDaily(ctx context.Context, bookId string, score int) error
	GetBookThermalWeekly(ctx context.Context, page int, size int) ([]string, error)
	GetBookThermalMonthly(ctx context.Context, page int, size int) ([]string, error)
}

type bookCache struct {
	redis *redis.Client
}

func NewBookCatch(redis *redis.Client) BookCache {
	return &bookCache{redis: redis}
}

func KeyBook(bookId string) string {
	return fmt.Sprintf(keyBook, bookId)
}

func (c *bookCache) GetBook(ctx context.Context, id string) (*model.Book, bool, error) {
	bookRes, err := c.redis.Get(ctx, KeyBook(id)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var bookDecode model.Book
	err = json.Unmarshal([]byte(bookRes), &bookDecode)
	if err != nil {
		return nil, false, err
	}
	return &bookDecode, true, nil
}

func (c *bookCache) UpdateBook(ctx context.Context, book *model.Book) error {
	bookEncode, err := json.Marshal(book)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, KeyBook(book.ID), bookEncode, time.Hour).Err()
}

func (c *bookCache) DeleteBook(ctx context.Context, id string) error {
	return c.redis.Del(ctx, KeyBook(id)).Err()
}

func (c *bookCache) GetBookThermalDaily(ctx context.Context, page int, size int) ([]string, error) {
	rankOffsetStart := int64((page - 1) * size)
	rankOffsetEnd := int64(page*size - 1)
	var rankRes *redis.StringSliceCmd
	rankOffsetExist := c.redis.Exists(ctx, constant.RedisBookThermalDailyTotal).Val()
	if rankOffsetExist == 1 {
		rankRes = c.redis.ZRangeArgs(ctx, redis.ZRangeArgs{
			Key:   constant.RedisBookThermalDailyTotal,
			Start: rankOffsetStart,
			Stop:  rankOffsetEnd,
			Rev:   true,
		})
	} else {
		// 当整合排行榜缓存失效时
		unionStore := &redis.ZStore{
			Keys: []string{
				constant.RedisBookThermalDaily(time.Now()),
				constant.RedisBookThermalDaily(time.Now().Add(-24 * time.Hour)),
			},
		}
		unionStoreRes := c.redis.ZUnionStore(ctx, constant.RedisBookThermalDailyTotal, unionStore)
		if unionStoreRes.Err() != nil {
			return nil, unionStoreRes.Err()
		}
		rankRes = c.redis.ZRangeArgs(ctx, redis.ZRangeArgs{
			Key:   constant.RedisBookThermalDailyTotal,
			Start: rankOffsetStart,
			Stop:  rankOffsetEnd,
			Rev:   true,
		})

		// 设置排行榜缓存时间
		c.redis.Expire(ctx, constant.RedisBookThermalDailyTotal, 5*time.Minute)

		// 持久化到排行榜历史
		copyToday := redis.ZStore{
			Keys: []string{constant.RedisBookThermalDaily(time.Now())},
		}
		_ = c.redis.ZUnionStore(ctx, constant.RedisBookThermalDay(time.Now()), &copyToday)
	}
	if rankRes.Err() != nil {
		return nil, rankRes.Err()
	}
	if len(rankRes.Val()) == 0 {
		return make([]string, 0), nil
	}
	return rankRes.Val(), nil
}

func (c *bookCache) IncreaseBookThermalDaily(ctx context.Context, bookId string, score int) error {
	// 如果排行榜没有字段则创建书籍
	bookRankErr := c.redis.ZScore(ctx, constant.RedisBookThermalDaily(time.Now()), bookId).Err()
	if bookRankErr != nil && bookRankErr != redis.Nil {
		return bookRankErr
	}
	if bookRankErr == redis.Nil {
		bookshelfArgs := redis.Z{Score: 0, Member: bookId}
		c.redis.ZAdd(ctx, constant.RedisBookThermalDaily(time.Now()), bookshelfArgs)
	}

	// 排行榜增加权值
	incrBookScoreErr := c.redis.ZIncrBy(ctx, constant.RedisBookThermalDaily(time.Now()), float64(score), bookId).Err()
	if incrBookScoreErr != nil {
		return incrBookScoreErr
	}

	// 如果没有过期时间则设置过期时间
	expRes := c.redis.TTL(ctx, constant.RedisBookThermalDaily(time.Now()))
	if expRes.Err() != nil {
		logger.Error(fmt.Sprintf("redis:set book thermal daily ttl fail:%v", expRes.Err()))
		return expRes.Err()
	}
	if expRes.Val() == -1 {
		c.redis.Expire(ctx, constant.RedisBookThermalDaily(time.Now()), time.Duration(3*24*time.Hour))
	}
	return nil
}

func (c *bookCache) GetBookThermalWeekly(ctx context.Context, page int, size int) ([]string, error) {
	existKey, err := c.redis.Exists(ctx, constant.RedisBookThermalWeeklyTotal).Result()
	if err != nil {
		return nil, err
	}
	if existKey != 1 {
		// 创建排行
		weekStore := redis.ZStore{
			Keys: []string{
				constant.RedisBookThermalDay(time.Now()),
				constant.RedisBookThermalDay(time.Now().Add(-1 * 24 * time.Hour)),
				constant.RedisBookThermalDay(time.Now().Add(-2 * 24 * time.Hour)),
				constant.RedisBookThermalDay(time.Now().Add(-3 * 24 * time.Hour)),
				constant.RedisBookThermalDay(time.Now().Add(-4 * 24 * time.Hour)),
				constant.RedisBookThermalDay(time.Now().Add(-5 * 24 * time.Hour)),
				constant.RedisBookThermalDay(time.Now().Add(-6 * 24 * time.Hour)),
			},
		}
		err := c.redis.ZUnionStore(ctx, constant.RedisBookThermalWeeklyTotal, &weekStore).Err()
		if err != nil {
			return nil, err
		}

		_ = c.redis.Expire(ctx, constant.RedisBookThermalWeeklyTotal, time.Duration(30*24*time.Hour)).Err()
	}

	startOffset := int64((page - 1) * size)
	endOffset := int64(page*size - 1)
	rankArgs := redis.ZRangeArgs{
		Key:   constant.RedisBookThermalWeeklyTotal,
		Start: startOffset,
		Stop:  endOffset,
		Rev:   true,
	}

	rank, err := c.redis.ZRangeArgs(ctx, rankArgs).Result()
	if err != nil {
		return nil, err
	}
	return rank, nil
}

func (c *bookCache) GetBookThermalMonthly(ctx context.Context, page int, size int) ([]string, error) {
	existKey, err := c.redis.Exists(ctx, constant.RedisBookThermalMonthlyTotal).Result()
	if err != nil {
		return nil, err
	}
	if existKey != 1 {
		// 创建排行
		monthKeys := make([]string, 0)
		for i := 0; i <= 30; i++ {
			keyTime := time.Now().Add(-time.Duration(i) * 24 * time.Hour)
			monthKeys = append(monthKeys, constant.RedisBookThermalDay(keyTime))
		}

		monthStore := redis.ZStore{
			Keys: monthKeys,
		}

		err := c.redis.ZUnionStore(ctx, constant.RedisBookThermalMonthlyTotal, &monthStore).Err()
		if err != nil {
			return nil, err
		}

		_ = c.redis.Expire(ctx, constant.RedisBookThermalMonthlyTotal, time.Duration(30*24*time.Hour)).Err()
	}

	startOffset := int64((page - 1) * size)
	endOffset := int64(page*size - 1)
	rankArgs := redis.ZRangeArgs{
		Key:   constant.RedisBookThermalMonthlyTotal,
		Start: startOffset,
		Stop:  endOffset,
		Rev:   true,
	}

	rank, err := c.redis.ZRangeArgs(ctx, rankArgs).Result()
	if err != nil {
		return nil, err
	}
	return rank, nil
}
