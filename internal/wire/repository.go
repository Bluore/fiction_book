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
