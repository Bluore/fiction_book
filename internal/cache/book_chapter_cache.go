package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/model"
	"fiction-book/pkg/constant"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	keyBookChapter = "GSF:book:cache:chapter:%s:detail"

	keyBookChapterLock = "GSF:book:lock:chapter:%s:lock"

	timeBookChapter = time.Hour
)

func KeyBookChapter(chapterId string) string {
	return fmt.Sprintf(keyBookChapter, chapterId)
}

func KeyBookChapterLock(chapterID string) string {
	return fmt.Sprintf(keyBookChapterLock, chapterID)
}

func TimeBookChapter() time.Duration {
	return timeBookChapter + TimeShakeDuration(timeBookChapter)
}

type BookChapterCache interface {
	UpdateBookChapter(ctx context.Context, chapterID string, chapter *model.BookChapter) error
	GetBookChapter(ctx context.Context, chapter string) (*model.BookChapter, bool, error)

	LockBookChapter(ctx context.Context, chapterID string) (string, bool, error)
	UnlockBookChapter(ctx context.Context, chapterID string, token string) error
}

type bookChapterCache struct {
	redis *redis.Client
}

func NewBookChapterCatch(redis *redis.Client) BookChapterCache {
	return &bookChapterCache{redis: redis}
}

func (c bookChapterCache) UpdateBookChapter(ctx context.Context, chapterID string, chapter *model.BookChapter) error {
	var chapterEncode []byte
	var err error
	if chapter != nil {
		chapterEncode, err = json.Marshal(chapter)
		if err != nil {
			return err
		}
	} else {
		chapterEncode = []byte(constant.RedisValueEmpty)
	}
	return c.redis.Set(ctx, KeyBookChapter(chapterID), chapterEncode, TimeBookChapter()).Err()
}

func (c bookChapterCache) GetBookChapter(ctx context.Context, chapter string) (*model.BookChapter, bool, error) {
	chapterRes, err := c.redis.Get(ctx, KeyBookChapter(chapter)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if chapterRes == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var chapterDecode model.BookChapter
	err = json.Unmarshal([]byte(chapterRes), &chapterDecode)
	if err != nil {
		return nil, false, err
	}
	return &chapterDecode, true, nil
}

func (c bookChapterCache) LockBookChapter(ctx context.Context, chapterID string) (string, bool, error) {
	token := uuid.New().String()
	result, err := c.redis.SetArgs(ctx, KeyBookChapterLock(chapterID), token, redis.SetArgs{
		Mode: "NX",
		TTL:  5 * time.Second,
	}).Result()
	if err != nil {
		return "", false, err
	}
	if result != "OK" {
		return "", false, fmt.Errorf("unlock book lock fail: error token: %s", token)
	}
	return token, true, nil
}

func (c bookChapterCache) UnlockBookChapter(ctx context.Context, chapterID string, token string) error {
	delKeyScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return -1
`)
	result, err := delKeyScript.Run(
		ctx,
		c.redis,
		[]string{KeyBookChapter(chapterID)},
		token,
	).Result()
	if err != nil {
		return err
	}
	if result != "1" {
		return fmt.Errorf("unlock book chapter lock fail: error token: %s", token)
	}
	return nil
}
