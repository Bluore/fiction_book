package wire

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"fiction-book/internal/repository"
)

// RepositorySet 仓储层 Provider 集合
var RepositorySet = wire.NewSet(
	ProvideUserRepository,
	ProvideBookRepository,
	ProvideBookChapterRepository,
	ProvideUserAssetRepository,
	ProvideUserPurchasedBookRepository,
	ProvideUserPurchasedChapterRepository,
	ProvideUserBookshelfRepository,
)

// ProvideUserRepository 提供用户仓储
func ProvideUserRepository(db *gorm.DB) repository.UserRepository {
	return repository.NewUserRepository(db)
}

// ProvideBookRepository 提供书籍仓储
func ProvideBookRepository(db *gorm.DB) repository.BookRepository {
	return repository.NewBookRepository(db)
}

// ProvideBookChapterRepository 提供书籍章节仓库
func ProvideBookChapterRepository(db *gorm.DB) repository.BookChapterRepository {
	return repository.NewBookChapterRepository(db)
}

// ProvideUserAssetRepository 提供用户资产仓库
func ProvideUserAssetRepository(db *gorm.DB) repository.UserAssetRepository {
	return repository.NewUserAssetRepository(db)
}

// ProvideUserPurchasedBookRepository 提供用户已购买的书籍仓库
func ProvideUserPurchasedBookRepository(db *gorm.DB) repository.UserPurchasedBookRepository {
	return repository.NewUserPurchasedBookRepository(db)
}

// ProvideUserPurchasedChapterRepository 提供用户已购买的章节仓库
func ProvideUserPurchasedChapterRepository(db *gorm.DB) repository.UserPurchasedChapterRepository {
	return repository.NewUserPurchasedChapterRepository(db)
}

// ProvideUserBookshelfRepository 提供用户书架仓库
func ProvideUserBookshelfRepository(db *gorm.DB) repository.UserBookshelfRepository {
	return repository.NewUserBookshelfRepository(db)
}
