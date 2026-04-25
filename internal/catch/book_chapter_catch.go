package catch

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	keyBookChapter = "Catch:Chapter:%s"
)

func KeyBookChapter(chapterId string) string {
	return fmt.Sprintf(keyBookChapter, chapterId)
}

type BookChapterCatch interface {
	UpdateBookChapter(ctx context.Context, chapter *model.BookChapter) error
	GetBookChapter(ctx context.Context, chapter string) (*model.BookChapter, bool, error)
}

type bookChapterCatch struct {
	redis *redis.Client
}

func NewBookChapterCatch(redis *redis.Client) BookChapterCatch {
	return &bookChapterCatch{redis: redis}
}

func (c bookChapterCatch) UpdateBookChapter(ctx context.Context, chapter *model.BookChapter) error {
	chapterEncode, err := json.Marshal(chapter)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, KeyBookChapter(chapter.ID), chapterEncode, time.Hour).Err()
}

func (c bookChapterCatch) GetBookChapter(ctx context.Context, chapter string) (*model.BookChapter, bool, error) {
	chapterRes, err := c.redis.Get(ctx, fmt.Sprintf(keyBook, chapter)).Result()
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
