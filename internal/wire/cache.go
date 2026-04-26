package wire

import (
	"fiction-book/internal/cache"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// CatchSet 创建缓存层依赖注入
var CatchSet = wire.NewSet(
	ProvideBookCatch,
	ProvideBookChapterCatch,
	ProvideLimitCatch,
	ProvideUserStatCache,
)

// ProvideBookCatch 创建Book缓存仓库
func ProvideBookCatch(redis *redis.Client) cache.BookCache {
	return cache.NewBookCatch(redis)
}

// ProvideBookChapterCatch 创建BookChapter缓存仓库
func ProvideBookChapterCatch(redis *redis.Client) cache.BookChapterCache {
	return cache.NewBookChapterCatch(redis)
}

// ProvideLimitCatch 创建限流仓库
func ProvideLimitCatch(redis *redis.Client) cache.LimitCache {
	return cache.NewLimitCatch(redis)
}

// ProvideUserStatCache 创建用户信息仓库
func ProvideUserStatCache(redis *redis.Client) cache.UserStatCache {
	return cache.NewUserStatCache(redis)
}
