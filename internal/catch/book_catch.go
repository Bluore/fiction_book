package catch

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
	keyBook = "Catch:Book:%s"
)

type BookCatch interface {
	GetBookCatch(ctx context.Context, id string) (*model.Book, bool, error)
	UpdateBookCatch(ctx context.Context, book *model.Book) error
	DeleteBookCatch(ctx context.Context, id string) error
	GetBookThermalDailyCatch(ctx context.Context, page int, size int) ([]string, error)
	IncreaseBookThermalDailyCatch(ctx context.Context, bookId string, score int) error
}

type bookCatch struct {
	redis *redis.Client
}

func NewBookCatch(redis *redis.Client) BookCatch {
	return &bookCatch{redis: redis}
}

func KeyBook(bookId string) string {
	return fmt.Sprintf(keyBook, bookId)
}

func (c *bookCatch) GetBookCatch(ctx context.Context, id string) (*model.Book, bool, error) {
	bookRes, err := c.redis.Get(ctx, fmt.Sprintf(keyBook, id)).Result()
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

func (c *bookCatch) UpdateBookCatch(ctx context.Context, book *model.Book) error {
	bookEncode, err := json.Marshal(book)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, KeyBook(book.ID), bookEncode, time.Hour).Err()
}

func (c *bookCatch) DeleteBookCatch(ctx context.Context, id string) error {
	return c.redis.Del(ctx, KeyBook(id)).Err()
}

func (c *bookCatch) GetBookThermalDailyCatch(ctx context.Context, page int, size int) ([]string, error) {
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
	}
	if rankRes.Err() != nil {
		return nil, rankRes.Err()
	}
	if len(rankRes.Val()) == 0 {
		return make([]string, 0), nil
	}
	return rankRes.Val(), nil
}

func (c *bookCatch) IncreaseBookThermalDailyCatch(ctx context.Context, bookId string, score int) error {
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
