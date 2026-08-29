//go:build wireinject
// +build wireinject

package wire

import (
	"fiction-book/internal/api/handler"
	"fiction-book/internal/kafka"
	"fiction-book/internal/server"

	"github.com/google/wire"
)

// InitializeApp 初始化整个应用
func InitializeApp() (*App, error) {
	wire.Build(
		// 基础设施层
		InfraSet,

		// 仓储层
		RepositorySet,

		// 缓存层
		CatchSet,

		// MQ层
		MassageQueueSet,

		// 服务层
		ServiceSet,

		// 后台服务层
		ServerSet,

		// Handler 层
		HandlerSet,

		// App 结构
		NewApp,
	)
	return nil, nil
}

// App 应用容器
type App struct {
	Handler *handler.Handler
	Kafka   *kafka.Handler
	Server  *server.Handle
}

// NewApp 创建 App 实例
func NewApp(h *handler.Handler, k *kafka.Handler, s *server.Handle) *App {
	return &App{Handler: h, Kafka: k, Server: s}
}
