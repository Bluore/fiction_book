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
	keyBookChapterInfo    = "GSF:book:cache:chapter:%s:info:detail"
	keyBookChapterContent = "GSF:book:cache:chapter:%s:content:detail"

	keyBookChapterInfoLock    = "GSF:book:lock:chapter:%s:info:lock"
	keyBookChapterContentLock = "GSF:book:lock:chapter:%s:content:lock"

	timeBookChapterInfo    = time.Hour
	timeBookChapterContent = time.Hour
)

func KeyBookChapterInfo(chapterId string) string {
	return fmt.Sprintf(keyBookChapterInfo, chapterId)
}

func KeyBookChapterContent(chapterId string) string {
	return fmt.Sprintf(keyBookChapterContent, chapterId)
}

func KeyBookChapterInfoLock(chapterID string) string {
	return fmt.Sprintf(keyBookChapterInfoLock, chapterID)
}

func KeyBookChapterContentLock(chapterID string) string {
	return fmt.Sprintf(keyBookChapterContentLock, chapterID)
}

func TimeBookChapterInfo() time.Duration {
	return timeBookChapterInfo + TimeShakeDuration(timeBookChapterInfo)
}

func TimeBookChapterContent() time.Duration {
	return timeBookChapterContent + TimeShakeDuration(timeBookChapterContent)
}

type BookChapterCache interface {
	UpdateBookChapterInfo(ctx context.Context, chapterID string, chapterInfo *model.BookChapterInfo) error
	UpdateBookChapterContent(ctx context.Context, chapterID string, chapterContent *model.BookChapterContent) error
	GetBookChapterInfo(ctx context.Context, chapterID string) (*model.BookChapterInfo, bool, error)
	GetBookChapterContent(ctx context.Context, chapterID string) (*model.BookChapterContent, bool, error)

	LockBookChapterInfo(ctx context.Context, chapterID string) (string, bool, error)
	LockBookChapterContent(ctx context.Context, chapterID string) (string, bool, error)
	UnlockBookChapterInfo(ctx context.Context, chapterID string, token string) error
	UnlockBookChapterContent(ctx context.Context, chapterID string, token string) error
}

type bookChapterCache struct {
	redis *redis.Client
}

func NewBookChapterCatch(redis *redis.Client) BookChapterCache {
	return &bookChapterCache{redis: redis}
}

// UpdateBookChapterInfo 更新章节信息缓存
func (c bookChapterCache) UpdateBookChapterInfo(ctx context.Context, chapterID string, chapterInfo *model.BookChapterInfo) error {
	var chapterEncode []byte
	var err error
	if chapterInfo != nil {
		chapterEncode, err = json.Marshal(chapterInfo)
		if err != nil {
			return err
		}
	} else {
		chapterEncode = []byte(constant.RedisValueEmpty)
	}
	return c.redis.Set(ctx, KeyBookChapterInfo(chapterID), chapterEncode, TimeBookChapterInfo()).Err()
}

// UpdateBookChapterContent 更新章节内容缓存
func (c bookChapterCache) UpdateBookChapterContent(ctx context.Context, chapterID string, chapterContent *model.BookChapterContent) error {
	var contentEncode []byte
	var err error
	if chapterContent != nil {
		contentEncode, err = json.Marshal(chapterContent)
		if err != nil {
			return err
		}
	} else {
		contentEncode = []byte(constant.RedisValueEmpty)
	}
	return c.redis.Set(ctx, KeyBookChapterContent(chapterID), contentEncode, TimeBookChapterContent()).Err()
}

// GetBookChapterInfo 获取章节信息缓存
func (c bookChapterCache) GetBookChapterInfo(ctx context.Context, chapterID string) (*model.BookChapterInfo, bool, error) {
	chapterRes, err := c.redis.Get(ctx, KeyBookChapterInfo(chapterID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if chapterRes == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var chapterDecode model.BookChapterInfo
	err = json.Unmarshal([]byte(chapterRes), &chapterDecode)
	if err != nil {
		return nil, false, err
	}
	return &chapterDecode, true, nil
}

// GetBookChapterContent 获取章节内容缓存
func (c bookChapterCache) GetBookChapterContent(ctx context.Context, chapterID string) (*model.BookChapterContent, bool, error) {
	contentRes, err := c.redis.Get(ctx, KeyBookChapterContent(chapterID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if contentRes == constant.RedisValueEmpty {
		return nil, true, nil
	}

	var contentDecode model.BookChapterContent
	err = json.Unmarshal([]byte(contentRes), &contentDecode)
	if err != nil {
		return nil, false, err
	}
	return &contentDecode, true, nil
}

// LockBookChapterInfo 锁定章节信息
func (c bookChapterCache) LockBookChapterInfo(ctx context.Context, chapterID string) (string, bool, error) {
	token := uuid.New().String()
	result, err := c.redis.SetArgs(ctx, KeyBookChapterInfoLock(chapterID), token, redis.SetArgs{
		Mode: "NX",
		TTL:  5 * time.Second,
	}).Result()
	if err != nil {
		return "", false, err
	}
	if result != "OK" {
		return "", false, fmt.Errorf("lock book_chapter_info fail: error token: %s", token)
	}
	return token, true, nil
}

// LockBookChapterContent 锁定章节内容
func (c bookChapterCache) LockBookChapterContent(ctx context.Context, chapterID string) (string, bool, error) {
	token := uuid.New().String()
	result, err := c.redis.SetArgs(ctx, KeyBookChapterContentLock(chapterID), token, redis.SetArgs{
		Mode: "NX",
		TTL:  5 * time.Second,
	}).Result()
	if err != nil {
		return "", false, err
	}
	if result != "OK" {
		return "", false, fmt.Errorf("lock book_chapter_content fail: error token: %s", token)
	}
	return token, true, nil
}

// UnlockBookChapterInfo 解锁章节信息
func (c bookChapterCache) UnlockBookChapterInfo(ctx context.Context, chapterID string, token string) error {
	delKeyScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return -1
`)
	result, err := delKeyScript.Run(
		ctx,
		c.redis,
		[]string{KeyBookChapterInfoLock(chapterID)},
		token,
	).Result()
	if err != nil {
		return err
	}
	if result != int64(1) {
		return fmt.Errorf("unlock book_chapter_info lock fail: error token: %s", token)
	}
	return nil
}

// UnlockBookChapterContent 解锁章节内容
func (c bookChapterCache) UnlockBookChapterContent(ctx context.Context, chapterID string, token string) error {
	delKeyScript := redis.NewScript(`
if redis.call('get', KEYS[1]) == ARGV[1] then
	return redis.call('del', KEYS[1])
end
return -1
`)
	result, err := delKeyScript.Run(
		ctx,
		c.redis,
		[]string{KeyBookChapterContentLock(chapterID)},
		token,
	).Result()
	if err != nil {
		return err
	}
	if result != int64(1) {
		return fmt.Errorf("unlock book_chapter_content lock fail: error token: %s", token)
	}
	return nil
}
