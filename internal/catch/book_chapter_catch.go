package catch

import "github.com/redis/go-redis/v9"

type BookChapterCatch interface {
}

type bookChapterCatch struct {
	redis *redis.Client
}

func NewBookChapterCatch(redis *redis.Client) BookChapterCatch {
	return &bookChapterCatch{redis: redis}
}
