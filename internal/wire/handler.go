package wire

import (
	"github.com/google/wire"
	
	"fiction-book/internal/api/handler"
	"fiction-book/internal/service"
)

// HandlerSet Handler 层 Provider 集合
var HandlerSet = wire.NewSet(
	ProvideHandler,
)

// ProvideHandler 提供主 Handler
func ProvideHandler(userService service.UserService) *handler.Handler {
	return handler.NewHandler(userService)
}
