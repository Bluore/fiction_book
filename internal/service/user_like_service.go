package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/response"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	userLikeOperatorCreate = "create"
	userLikeOperatorDelete = "delete"
)

// UserLikeService 用户关注服务接口
type UserLikeService interface {
	UpdateUserLike(ctx context.Context, likerID string, req *dto.UpdateUserLikeRequest) (*dto.UpdateUserLikeResponse, error)
	GetUserLike(ctx context.Context, likerID string, req *dto.GetUserLikeRequest) (*dto.GetUserLikeResponse, error)
}

type userLikeService struct {
	userRepo      repository.UserRepository
	userLikeRepo  repository.UserLikeRepository
	userLikeCache cache.UserLikeCache
	db            *gorm.DB
}

// NewUserLikeService 创建用户关注服务实例
func NewUserLikeService(
	userRepo repository.UserRepository,
	userLikeRepo repository.UserLikeRepository,
	userLikeCache cache.UserLikeCache,
	db *gorm.DB,
) UserLikeService {
	return &userLikeService{
		userRepo:      userRepo,
		userLikeRepo:  userLikeRepo,
		userLikeCache: userLikeCache,
		db:            db,
	}
}

func (s userLikeService) UpdateUserLike(ctx context.Context, likerID string, req *dto.UpdateUserLikeRequest) (*dto.UpdateUserLikeResponse, error) {
	switch req.Operator {
	case userLikeOperatorCreate, userLikeOperatorDelete:
	default:
		return nil, response.ErrParam
	}

	// 先判断目标用户是否存在
	targetUserDetail, err := s.userRepo.GetByID(ctx, s.db, req.TargetID)
	if err != nil {
		return nil, err
	}
	if targetUserDetail == nil {
		return nil, response.ErrUserNotFound
	}

	// UserID 为被关注用户，LikerID 为发起关注的用户
	userLikeDetail := &model.UserLike{
		ID:      uuid.New().String(),
		UserID:  req.TargetID,
		LikerID: likerID,
	}

	switch req.Operator {
	case userLikeOperatorCreate:
		err = createUserLikeDetailWithCache(ctx, s.db, s.userLikeCache, s.userLikeRepo, userLikeDetail)
		if err != nil {
			return nil, err
		}
		return &dto.UpdateUserLikeResponse{
			TargetID: req.TargetID,
			IsLike:   true,
		}, nil
	case userLikeOperatorDelete:
		err = deleteUserLikeDetailWithCache(ctx, s.db, s.userLikeCache, s.userLikeRepo, userLikeDetail)
		if err != nil {
			return nil, err
		}
		return &dto.UpdateUserLikeResponse{
			TargetID: req.TargetID,
			IsLike:   false,
		}, nil
	}
	return nil, response.ErrParam
}

func (s userLikeService) GetUserLike(ctx context.Context, likerID string, req *dto.GetUserLikeRequest) (*dto.GetUserLikeResponse, error) {
	// 先判断目标用户是否存在
	target, err := s.userRepo.GetByID(ctx, s.db, req.TargetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, response.ErrUserNotFound
	}

	userLike, err := queryUserLikeDetailWithCache(ctx, s.db, s.userLikeCache, s.userLikeRepo, req.TargetID, likerID)
	if err != nil {
		return nil, err
	}
	return &dto.GetUserLikeResponse{TargetID: req.TargetID, IsLike: userLike != nil}, nil
}
