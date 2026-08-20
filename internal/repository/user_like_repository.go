package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrUserLikeAlreadyExists = errors.New("user like already exists")

type UserLikeRepository interface {
	GetByUserIDAndLikerID(ctx context.Context, db *gorm.DB, userID string, likerID string) (*model.UserLike, error)
	CountByUserID(ctx context.Context, db *gorm.DB, userID string) (int, error)
	Create(ctx context.Context, db *gorm.DB, userLike *model.UserLike) error
	Update(ctx context.Context, db *gorm.DB, userLike *model.UserLike) error
	DeleteByUserIDAndLikerID(ctx context.Context, db *gorm.DB, userID string, likerID string) error
}

type userLikeRepository struct {
	db *gorm.DB
}

func NewUserLikeRepository(db *gorm.DB) UserLikeRepository {
	return &userLikeRepository{
		db: db,
	}
}

func (r userLikeRepository) GetByUserIDAndLikerID(ctx context.Context, db *gorm.DB, userID string, likerID string) (*model.UserLike, error) {
	var userLike model.UserLike
	err := db.WithContext(ctx).
		Where("user_id = ? AND liker_id = ?", userID, likerID).
		First(&userLike).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &userLike, nil
}

func (r userLikeRepository) CountByUserID(ctx context.Context, db *gorm.DB, userID string) (int, error) {
	var count int64
	err := db.WithContext(ctx).
		Model(&model.UserLike{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return int(count), err
}

func (r userLikeRepository) Create(ctx context.Context, db *gorm.DB, userLike *model.UserLike) error {
	if userLike.ID == "" {
		userLike.ID = uuid.NewString()
	}
	return db.WithContext(ctx).Create(userLike).Error
}

func (r userLikeRepository) Update(ctx context.Context, db *gorm.DB, userLike *model.UserLike) error {
	return db.WithContext(ctx).Save(userLike).Error
}

func (r userLikeRepository) DeleteByUserIDAndLikerID(ctx context.Context, db *gorm.DB, userID string, likerID string) error {
	return db.WithContext(ctx).
		Where("user_id = ? AND liker_id = ?", userID, likerID).
		Delete(&model.UserLike{}).Error
}
