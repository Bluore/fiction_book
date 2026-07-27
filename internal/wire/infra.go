package wire

import (
	"fiction-book/pkg/logger"
	"fiction-book/pkg/payment"
	"fiction-book/pkg/redisclient"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"fiction-book/pkg/config"
	"fiction-book/pkg/database"
)

// InfraSet 基础设施 Provider 集合
var InfraSet = wire.NewSet(
	ProvideConfig,
	ProvideDatabase,
	ProvideCatch,
	ProvideESClient,
	ProvideAlipayStrategy,
	ProvideAgilePool,
)

// ProvideConfig 提供配置
func ProvideConfig() (*config.Config, error) {
	return config.Load()
}

// ProvideDatabase 提供数据库连接
func ProvideDatabase(cfg *config.Config) (*gorm.DB, error) {
	return database.InitDB(cfg)
}

// ProvideCatch 提供缓存数据库链接
func ProvideCatch(cfg *config.Config) (*redis.Client, error) {
	if err := redisclient.Init(&cfg.Redis); err != nil {
		logger.Fatal("init redis error", zap.Error(err))
	}
	return redisclient.GetClient()
}

// ProvideESClient 提供 Elasticsearch 客户端链接
func ProvideESClient(cfg *config.Config) (*elasticsearch.Client, error) {
	return database.InitES(cfg)
}

// ProvideAlipayStrategy 提供支付宝支付策略
func ProvideAlipayStrategy(cfg *config.Config) (payment.Strategy, error) {
	return payment.NewAlipayStrategy(cfg.Payment.Alipay)
}

// ProvideAgilePool 提供线程池
func ProvideAgilePool(cfg *config.Config) (*agilePool.Pool, error) {
	return agilePool.NewPool(
		agilePool.NewConfig(
			agilePool.WithBlockMode(agilePool.BLOCK),
			agilePool.WithTaskQueueSize(10000),
		),
	), nil
}
