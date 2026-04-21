package wire

import (
	"fiction-book/pkg/redisclient"
	"log"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"fiction-book/pkg/config"
	"fiction-book/pkg/database"
)

// InfraSet 基础设施 Provider 集合
var InfraSet = wire.NewSet(
	ProvideConfig,
	ProvideDatabase,
	ProvideCatch,
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
		log.Fatal(err)
	}
	return redisclient.GetClient()
}
