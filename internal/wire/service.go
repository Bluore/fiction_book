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
	ProvideBookChapterService,
)

// ProvideUserService 提供用户服务
func ProvideUserService(
	userRepo repository.UserRepository,
	cfg *config.Config,
) service.UserService {
	return service.NewUserService(userRepo, cfg)
}

// ProvideBookService 提供书籍服务
func ProvideBookService(
	bookRepo repository.BookRepository,
	cfg *config.Config,
) service.BookService {
	return service.NewBookService(bookRepo, cfg)
}

func ProvideBookChapterService(
	bookRepo repository.BookRepository,
	bookChapterRepo repository.BookChapterRepository,
	cfg *config.Config,
) service.BookChapterService {
	return service.NewBookChapterService(bookRepo, bookChapterRepo, cfg)
}
