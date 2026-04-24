package repository

import (
	"context"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
)

type UserPurchasedChapterRepository interface {
	GetListByUserID(ctx context.Context, userID string) (*[]model.UserPurchasedChapter, error)
	CheckPurchasedChapter(ctx context.Context, userID, chapterID string) (bool, *model.UserPurchasedChapter, error)
	AddPurchasedChapter(ctx context.Context, in *model.UserPurchasedChapter) error
}

type userPurchasedChapterRepository struct {
	db *gorm.DB
}

func NewUserPurchasedChapterRepository(db *gorm.DB) UserPurchasedChapterRepository {
	return &userPurchasedChapterRepository{
		db: db,
	}
}

func (r userPurchasedChapterRepository) GetListByUserID(ctx context.Context, userID string) (*[]model.UserPurchasedChapter, error) {
	var chapters []model.UserPurchasedChapter
	err := r.db.Where("active_at > ?", time.Now()).Where("user_id = ?", userID).Find(&chapters).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &[]model.UserPurchasedChapter{}, nil
		}
		return nil, err
	}
	return &chapters, nil
}

func (r userPurchasedChapterRepository) CheckPurchasedChapter(ctx context.Context, userID, chapterID string) (bool, *model.UserPurchasedChapter, error) {
	var chapter model.UserPurchasedChapter
	err := r.db.Where("active_at > ?", time.Now()).Where("user_id = ? AND chapter_id = ?", userID, chapterID).First(&chapter).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &chapter, nil
}

func (r userPurchasedChapterRepository) AddPurchasedChapter(ctx context.Context, in *model.UserPurchasedChapter) error {
	return r.db.WithContext(ctx).Create(in).Error
}
