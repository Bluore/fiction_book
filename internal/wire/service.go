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
	ProvideUserAssetService,
	ProvideUserPurchasedBookService,
)

// ProvideUserService 提供用户服务
func ProvideUserService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	cfg *config.Config,
) service.UserService {
	return service.NewUserService(userRepo, userAssetRepo, cfg)
}

// ProvideBookService 提供书籍服务
func ProvideBookService(
	bookRepo repository.BookRepository,
	cfg *config.Config,
) service.BookService {
	return service.NewBookService(bookRepo, cfg)
}

// ProvideBookChapterService 提供章节仓库
func ProvideBookChapterService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	bookChapterRepo repository.BookChapterRepository,
	cfg *config.Config,
) service.BookChapterService {
	return service.NewBookChapterService(userRepo, bookRepo, bookChapterRepo, cfg)
}

// ProvideUserAssetService  提供用户资产仓库
func ProvideUserAssetService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	cfg *config.Config,
) service.UserAssetService {
	return service.NewUserAssetService(userRepo, userAssetRepo, cfg)
}

// ProvideUserPurchasedBookService  提供用户已购买的书籍仓库
func ProvideUserPurchasedBookService(
	userRepo repository.UserRepository,
	userPurchasedBookRepo repository.UserPurchasedBookRepository,
	cfg *config.Config,
) service.UserPurchasedBookService {
	return service.NewUserPurchasedBookService(userRepo, userPurchasedBookRepo, cfg)

}
