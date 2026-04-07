package repository

import (
	"context"
	"fiction-book/internal/model"
	"time"

	"gorm.io/gorm"
)

type UserPurchasedBookRepository interface {
	GetListByUserID(ctx context.Context, userID string) (*[]model.UserPurchasedBook, error)
	CheckPurchasedBook(ctx context.Context, userID, BookID string) (bool, *model.UserPurchasedBook, error)
}

type userPurchasedBookRepository struct {
	db *gorm.DB
}

func NewUserPurchasedBookRepository(db *gorm.DB) UserPurchasedBookRepository {
	return &userPurchasedBookRepository{
		db: db,
	}
}

func (r userPurchasedBookRepository) GetListByUserID(ctx context.Context, userID string) (*[]model.UserPurchasedBook, error) {
	var users []model.UserPurchasedBook
	err := r.db.Where("active_at > ?", time.Now()).Where("user_id = ?", userID).Find(&users).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &[]model.UserPurchasedBook{}, nil
		}
		return nil, err
	}
	return &users, nil
}

func (r userPurchasedBookRepository) CheckPurchasedBook(ctx context.Context, userID, BookID string) (bool, *model.UserPurchasedBook, error) {
	var user model.UserPurchasedBook
	err := r.db.Where("active_at > ?", time.Now()).Where("user_id = ? && book_id = ?", userID, BookID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &user, nil
}
