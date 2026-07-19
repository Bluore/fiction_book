//go:build wireinject
// +build wireinject

package wire

import (
	mq "fiction-book/internal/kafka"

	"github.com/google/wire"

	"fiction-book/internal/api/handler"
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

		// 服务层
		ServiceSet,

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
}

// NewApp 创建 App 实例
func NewApp(h *handler.Handler) *App {
	return &App{Handler: h}
}

func InitializeMassageQueue() (*MassageQueue, error) {
	wire.Build(
		InfraSet,
		RepositorySet,
		MassageHandlerSet,
		NewMassageQueue,
	)
	return nil, nil
}

type MassageQueue struct {
	Handler *mq.Handler
}

func NewMassageQueue(h *mq.Handler) *MassageQueue {
	return &MassageQueue{Handler: h}
}
