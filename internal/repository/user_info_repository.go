package repository

import (
	"context"
	"errors"

	"fiction-book/internal/model"

	"gorm.io/gorm"
)

// UserInfoRepository 用户详情仓储接口
type UserInfoRepository interface {
	Create(ctx context.Context, db *gorm.DB, userInfo *model.UserInfo) error
	GetByUserID(ctx context.Context, db *gorm.DB, userID string) (*model.UserInfo, error)
	GetByUsername(ctx context.Context, db *gorm.DB, username string) (*model.UserInfo, error)
	Update(ctx context.Context, db *gorm.DB, userInfo *model.UserInfo) error
	Delete(ctx context.Context, db *gorm.DB, userID string) error
	List(ctx context.Context, db *gorm.DB, offset, limit int) ([]*model.UserInfo, error)
}

type userInfoRepository struct {
	db *gorm.DB
}

// NewUserInfoRepository 创建用户详情仓储实例
func NewUserInfoRepository(db *gorm.DB) UserInfoRepository {
	return &userInfoRepository{db: db}
}

func (r *userInfoRepository) Create(ctx context.Context, db *gorm.DB, userInfo *model.UserInfo) error {
	return db.WithContext(ctx).Create(userInfo).Error
}

func (r *userInfoRepository) GetByUserID(ctx context.Context, db *gorm.DB, userID string) (*model.UserInfo, error) {
	var userInfo model.UserInfo
	err := db.WithContext(ctx).Where("user_id = ?", userID).First(&userInfo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}

func (r *userInfoRepository) GetByUsername(ctx context.Context, db *gorm.DB, username string) (*model.UserInfo, error) {
	var userInfo model.UserInfo
	err := db.WithContext(ctx).Where("username = ?", username).First(&userInfo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}

func (r *userInfoRepository) Update(ctx context.Context, db *gorm.DB, userInfo *model.UserInfo) error {
	return db.WithContext(ctx).Save(userInfo).Error
}

func (r *userInfoRepository) Delete(ctx context.Context, db *gorm.DB, userID string) error {
	return db.WithContext(ctx).Delete(&model.UserInfo{}, "user_id = ?", userID).Error
}

func (r *userInfoRepository) List(ctx context.Context, db *gorm.DB, offset, limit int) ([]*model.UserInfo, error) {
	var userInfos []*model.UserInfo
	err := db.WithContext(ctx).Offset(offset).Limit(limit).Find(&userInfos).Error
	return userInfos, err
}
