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
	UpdateUsername(ctx context.Context, db *gorm.DB, userID, username string) error
	UpdateProfileURL(ctx context.Context, db *gorm.DB, userID, profileURL string) error
	Delete(ctx context.Context, db *gorm.DB, userID string) error
	List(ctx context.Context, db *gorm.DB, offset, limit int) ([]*model.UserInfo, error)

	// 计数类字段的原子增加操作，delta 传负值表示减少
	IncrementReadBookNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error
	IncrementReadWordsNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error
	IncrementFansNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error
	IncrementLikeNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error
	IncrementBookShelfNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error
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

// UpdateUsername 只更新用户名字段，避免覆盖计数类字段或为历史缺失记录插入空行
func (r *userInfoRepository) UpdateUsername(ctx context.Context, db *gorm.DB, userID, username string) error {
	return db.WithContext(ctx).Model(&model.UserInfo{}).
		Where("user_id = ?", userID).
		Update("username", username).Error
}

// UpdateProfileURL 只更新头像地址字段，避免覆盖计数类字段或为历史缺失记录插入空行
func (r *userInfoRepository) UpdateProfileURL(ctx context.Context, db *gorm.DB, userID, profileURL string) error {
	return db.WithContext(ctx).Model(&model.UserInfo{}).
		Where("user_id = ?", userID).
		Update("user_profile_url", profileURL).Error
}

func (r *userInfoRepository) Delete(ctx context.Context, db *gorm.DB, userID string) error {
	return db.WithContext(ctx).Delete(&model.UserInfo{}, "user_id = ?", userID).Error
}

func (r *userInfoRepository) List(ctx context.Context, db *gorm.DB, offset, limit int) ([]*model.UserInfo, error) {
	var userInfos []*model.UserInfo
	err := db.WithContext(ctx).Offset(offset).Limit(limit).Find(&userInfos).Error
	return userInfos, err
}

func (r *userInfoRepository) IncrementReadBookNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error {
	return db.WithContext(ctx).Model(&model.UserInfo{}).
		Where("user_id = ?", userID).
		UpdateColumn("read_book_num", gorm.Expr("read_book_num + ?", delta)).Error
}

func (r *userInfoRepository) IncrementReadWordsNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error {
	return db.WithContext(ctx).Model(&model.UserInfo{}).
		Where("user_id = ?", userID).
		UpdateColumn("read_words_num", gorm.Expr("read_words_num + ?", delta)).Error
}

func (r *userInfoRepository) IncrementFansNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error {
	return db.WithContext(ctx).Model(&model.UserInfo{}).
		Where("user_id = ?", userID).
		UpdateColumn("fans_num", gorm.Expr("fans_num + ?", delta)).Error
}

func (r *userInfoRepository) IncrementLikeNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error {
	return db.WithContext(ctx).Model(&model.UserInfo{}).
		Where("user_id = ?", userID).
		UpdateColumn("like_num", gorm.Expr("like_num + ?", delta)).Error
}

func (r *userInfoRepository) IncrementBookShelfNum(ctx context.Context, db *gorm.DB, userID string, delta int64) error {
	return db.WithContext(ctx).Model(&model.UserInfo{}).
		Where("user_id = ?", userID).
		UpdateColumn("book_shelf_num", gorm.Expr("book_shelf_num + ?", delta)).Error
}
