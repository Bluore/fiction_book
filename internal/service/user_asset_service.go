package service

import (
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
)

type UserAssetService interface {
}

type userAssetService struct {
	userRepo      repository.UserRepository
	userAssetRepo repository.UserAssetRepository
	cfg           config.Config
}

func NewUserAssetService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	config *config.Config,
) UserAssetService {
	return &userAssetService{
		userRepo:      userRepo,
		userAssetRepo: userAssetRepo,
		cfg:           *config,
	}
}
