package wire

import (
	"github.com/google/wire"

	"github.com/d60-Lab/gin-template/internal/repository"
	"github.com/d60-Lab/gin-template/internal/service"
	"github.com/d60-Lab/gin-template/pkg/config"
)

// ServiceSet 服务层 Provider 集合
var ServiceSet = wire.NewSet(
	ProvideUserService,
)

// ProvideUserService 提供用户服务
func ProvideUserService(
	userRepo repository.UserRepository,
	cfg *config.Config,
) service.UserService {
	return service.NewUserService(userRepo, cfg)
}
