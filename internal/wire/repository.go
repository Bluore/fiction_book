package wire

import (
	"github.com/google/wire"
	"gorm.io/gorm"
	
	"fiction-book/internal/repository"
)

// RepositorySet 仓储层 Provider 集合
var RepositorySet = wire.NewSet(
	ProvideUserRepository,
)

// ProvideUserRepository 提供用户仓储
func ProvideUserRepository(db *gorm.DB) repository.UserRepository {
	return repository.NewUserRepository(db)
}
