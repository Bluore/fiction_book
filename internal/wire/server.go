package wire

import (
	"fiction-book/internal/cache"
	"fiction-book/internal/repository"
	"fiction-book/internal/server"
	"fiction-book/pkg/config"
	"fiction-book/pkg/payment"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ServerSet 后台服务层 Provider 集合
var ServerSet = wire.NewSet(
	ProvideOrderTempServer,
	ProvideServerHandle,
)

// ProvideOrderTempServer 提供订单临时后台服务
func ProvideOrderTempServer(
	orderRepo repository.OrderRepository,
	orderTempRepo repository.OrderTempRepository,
	userAssetRepo repository.UserAssetRepository,
	orderCache cache.OrderCache,
	userAssetCache cache.UserAssetCache,
	cfg *config.Config,
	db *gorm.DB,
	redis *redis.Client,
	pool *agilePool.Pool,
	payment payment.Strategy,
) *server.OrderTempServer {
	return server.NewOrderTempServer(
		orderRepo,
		orderTempRepo,
		userAssetRepo,
		orderCache,
		userAssetCache,
		cfg,
		db,
		redis,
		pool,
		payment,
	)
}

// ProvideServerHandle 提供后台服务聚合入口
func ProvideServerHandle(orderTempServer *server.OrderTempServer, cfg *config.Config) *server.Handle {
	return server.NewHandle(orderTempServer, cfg)
}
