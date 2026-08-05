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
	keyBookChapter = "GSF:book:cache:chapter:%s:detail"
)

func KeyBookChapter(chapterId string) string {
	return fmt.Sprintf(keyBookChapter, chapterId)
}

type BookChapterCache interface {
	UpdateBookChapter(ctx context.Context, chapter *model.BookChapter) error
	GetBookChapter(ctx context.Context, chapter string) (*model.BookChapter, bool, error)
}

type bookChapterCache struct {
	redis *redis.Client
}

func NewBookChapterCatch(redis *redis.Client) BookChapterCache {
	return &bookChapterCache{redis: redis}
}

func (c bookChapterCache) UpdateBookChapter(ctx context.Context, chapter *model.BookChapter) error {
	chapterEncode, err := json.Marshal(chapter)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, KeyBookChapter(chapter.ID), chapterEncode, time.Hour).Err()
}

func (c bookChapterCache) GetBookChapter(ctx context.Context, chapter string) (*model.BookChapter, bool, error) {
	chapterRes, err := c.redis.Get(ctx, KeyBookChapter(chapter)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var chapterDecode model.BookChapter
	err = json.Unmarshal([]byte(chapterRes), &chapterDecode)
	if err != nil {
		return nil, false, err
	}
	return &chapterDecode, true, nil
}
