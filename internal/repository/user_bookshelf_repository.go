package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserBookshelfRepository interface {
	ListByUserID(ctx context.Context, db *gorm.DB, userID string, page int, size int) ([]*model.UserBookshelf, error)
	ListUserIDsByBookID(ctx context.Context, db *gorm.DB, bookID string) ([]string, error)
	CreateByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) error
	DeleteByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) error
	CheckByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) (bool, *model.UserBookshelf, error)
	CountByUserID(ctx context.Context, db *gorm.DB, userID string) (int, error)
}

type userBookshelfRepository struct {
	db *gorm.DB
}

func NewUserBookshelfRepository(db *gorm.DB) UserBookshelfRepository {
	return &userBookshelfRepository{
		db: db,
	}
}

func (r userBookshelfRepository) ListByUserID(ctx context.Context, db *gorm.DB, userID string, page int, size int) ([]*model.UserBookshelf, error) {
	var books []*model.UserBookshelf
	err := db.WithContext(ctx).Where("user_id = ?", userID).Offset((page - 1) * size).Limit(size).Find(&books).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return make([]*model.UserBookshelf, 0), nil
		}
		return nil, err
	}
	return books, nil
}

func (r userBookshelfRepository) ListUserIDsByBookID(ctx context.Context, db *gorm.DB, bookID string) ([]string, error) {
	userIDs := make([]string, 0)
	err := db.WithContext(ctx).
		Model(&model.UserBookshelf{}).
		Where("book_id = ?", bookID).
		Order("user_id ASC").
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r userBookshelfRepository) CreateByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) error {
	var exist *model.UserBookshelf
	err := db.Where("user_id = ? AND book_id = ?", userID, bookID).First(&exist).Error
	if err == gorm.ErrRecordNotFound {
		// 不存在时再创建
		book := model.UserBookshelf{
			ID:     uuid.New().String(),
			BookID: bookID,
			UserID: userID,
		}
		return db.WithContext(ctx).Create(&book).Error
	}
	if err != nil {
		return err
	}
	return errors.New("bookshelf recoder exist")
}

func (r userBookshelfRepository) DeleteByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) error {
	//TODO implement me
	panic("implement me")
}

func (r userBookshelfRepository) CheckByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) (bool, *model.UserBookshelf, error) {
	//TODO implement me
	panic("implement me")
}

func (r userBookshelfRepository) CountByUserID(ctx context.Context, db *gorm.DB, userID string) (int, error) {
	var count int64
	err := db.WithContext(ctx).Model(&model.UserBookshelf{}).Where("user_id = ?", userID).Count(&count).Error
	return int(count), err
}
