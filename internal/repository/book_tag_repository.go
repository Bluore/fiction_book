package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"

	"gorm.io/gorm"
)

type BookTagRepository interface {
	ListByBookID(ctx context.Context, bookID string) ([]model.BookTag, error)
	DeleteAllByBookID(ctx context.Context, db *gorm.DB, bookID string) error
	CreateBatch(ctx context.Context, db *gorm.DB, tags []model.BookTag) error
}

type bookTagRepository struct {
	db *gorm.DB
}

func NewBookTagRepository(db *gorm.DB) BookTagRepository {
	return &bookTagRepository{db: db}
}

func (r *bookTagRepository) ListByBookID(ctx context.Context, bookID string) ([]model.BookTag, error) {
	tags := []model.BookTag{}
	err := r.db.WithContext(ctx).Where("book_id = ?", bookID).Find(&tags).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return make([]model.BookTag, 0), nil
		}
		return nil, err
	}
	return tags, nil
}

func (r *bookTagRepository) DeleteAllByBookID(ctx context.Context, db *gorm.DB, bookID string) error {
	return db.WithContext(ctx).Where("book_id = ?", bookID).Delete(&model.BookTag{}).Error
}

func (r *bookTagRepository) CreateBatch(ctx context.Context, db *gorm.DB, tags []model.BookTag) error {
	if len(tags) == 0 {
		return nil
	}
	return db.WithContext(ctx).Create(&tags).Error
}
