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
	ProvideUserAssetCache,
	ProvideAuthCache,
	ProvideOrderCache,
	ProvideBookThermalCache,
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

// ProvideUserAssetCache 创建用户资产缓存
func ProvideUserAssetCache(redis *redis.Client) cache.UserAssetCache {
	return cache.NewUserAssetCache(redis)
}

func ProvideAuthCache(redis *redis.Client) cache.AuthCache {
	return cache.NewAuthCache(redis)
}

// ProvideOrderCache 创建订单缓存
func ProvideOrderCache(redis *redis.Client) cache.OrderCache {
	return cache.NewOrderCache(redis)
}

// ProvideBookThermalCache 创建书籍热度缓存
func ProvideBookThermalCache(redis *redis.Client) cache.BookThermalCache {
	return cache.NewBookThermalCache(redis)
}
