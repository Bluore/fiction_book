package wire

import (
	"github.com/google/wire"

	"github.com/d60-Lab/gin-template/internal/api/handler"
	"github.com/d60-Lab/gin-template/internal/service"
)

// HandlerSet Handler 层 Provider 集合
var HandlerSet = wire.NewSet(
	ProvideHandler,
)

// ProvideHandler 提供主 Handler
func ProvideHandler(userService service.UserService) *handler.Handler {
	return handler.NewHandler(userService)
}
