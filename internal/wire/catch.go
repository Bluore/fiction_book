package wire

import (
	"fiction-book/internal/catch"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// CatchSet 创建缓存层依赖注入
var CatchSet = wire.NewSet(
	ProvideBookCatch,
	ProvideBookChapterCatch,
	ProvideLimitCatch,
)

// ProvideBookCatch 创建Book缓存仓库
func ProvideBookCatch(redis *redis.Client) catch.BookCatch {
	return catch.NewBookCatch(redis)
}

// ProvideBookChapterCatch 创建BookChapter缓存仓库
func ProvideBookChapterCatch(redis *redis.Client) catch.BookChapterCatch {
	return catch.NewBookChapterCatch(redis)
}

// ProvideLimitCatch 创建限流仓库
func ProvideLimitCatch(redis *redis.Client) catch.LimitCatch {
	return catch.NewLimitCatch(redis)
}
