package repository

import (
	"context"
	"fiction-book/internal/model"

	"gorm.io/gorm"
)

type UserHistoryRepository interface {
	ListByUserID(ctx context.Context, db *gorm.DB, userID string, page int, size int) ([]*model.UserHistory, error)
	CountByUserID(ctx context.Context, db *gorm.DB, userID string) (int, error)
}

type userHistoryRepository struct {
	db *gorm.DB
}

func NewUserHistoryRepository(db *gorm.DB) UserHistoryRepository {
	return &userHistoryRepository{
		db: db,
	}
}

func (r userHistoryRepository) ListByUserID(ctx context.Context, db *gorm.DB, userID string, page int, size int) ([]*model.UserHistory, error) {
	var books []*model.UserHistory
	err := db.WithContext(ctx).Where("user_id = ?", userID).Offset((page - 1) * size).Limit(size).Find(&books).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return make([]*model.UserHistory, 0), nil
		}
		return nil, err
	}
	return books, nil
}

func (r userHistoryRepository) CountByUserID(ctx context.Context, db *gorm.DB, userID string) (int, error) {
	var count int64
	err := db.WithContext(ctx).Model(&model.UserHistory{}).Where("user_id = ?", userID).Count(&count).Error
	return int(count), err
}
