package wire

import (
	"github.com/google/wire"

	"fiction-book/internal/repository"
	"fiction-book/internal/service"
	"fiction-book/pkg/config"
)

// ServiceSet 服务层 Provider 集合
var ServiceSet = wire.NewSet(
	ProvideUserService,
	ProvideBookService,
)

// ProvideUserService 提供用户服务
func ProvideUserService(
	userRepo repository.UserRepository,
	cfg *config.Config,
) service.UserService {
	return service.NewUserService(userRepo, cfg)
}

func ProvideBookService(
	bookRepo repository.BookRepository,
	cfg *config.Config,
) service.BookService {
	return service.NewBookService(bookRepo, cfg)
}
