package service

import (
	"context"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
)

type UserAssetService interface {
	GetByID(ctx context.Context, id string) (*dto.UserAssetResponse, error)
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

// GetByID 获取用户资产
func (u *userAssetService) GetByID(ctx context.Context, id string) (*dto.UserAssetResponse, error) {
	userAsset, err := u.userAssetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if userAsset == nil {
		return nil, ErrUserNotFound
	}

	return toUserAssetResponse(userAsset), nil
}

func toUserAssetResponse(userAsset *model.UserAsset) *dto.UserAssetResponse {
	return &dto.UserAssetResponse{
		ID:       userAsset.ID,
		Gold:     userAsset.Gold,
		VIPLevel: VIPStrToLevle(userAsset.VIPLevel),
		VIPMark:  userAsset.VIPLevel,
	}
}
