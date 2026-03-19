package wire

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"github.com/d60-Lab/gin-template/pkg/config"
	"github.com/d60-Lab/gin-template/pkg/database"
)

// InfraSet 基础设施 Provider 集合
var InfraSet = wire.NewSet(
	ProvideConfig,
	ProvideDatabase,
)

// ProvideConfig 提供配置
func ProvideConfig() (*config.Config, error) {
	return config.Load()
}

// ProvideDatabase 提供数据库连接
func ProvideDatabase(cfg *config.Config) (*gorm.DB, error) {
	return database.InitDB(cfg)
}
