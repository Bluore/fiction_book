package repository

import (
	"context"
	"errors"
	"fiction-book/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookChapterRepository interface {
	GetByID(ctx context.Context, id string) (*model.BookChapter, error)
	GetByBookIDAndOrder(ctx context.Context, id string, order int) (*model.BookChapter, error)
	GetIDListByBookID(ctx context.Context, id string) (*[]model.BookChapterInfo, error)
	AddByBookID(ctx context.Context, bookChapter *model.BookChapter) (*model.BookChapter, error)
	CountByBookID(ctx context.Context, bookID string) (int, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, bookChapter *model.BookChapter) error
}

type bookChapterRepository struct {
	db *gorm.DB
}

func NewBookChapterRepository(db *gorm.DB) BookChapterRepository {
	return &bookChapterRepository{db: db}
}

func (b bookChapterRepository) GetByID(ctx context.Context, id string) (*model.BookChapter, error) {
	var bookChapter model.BookChapter
	err := b.db.WithContext(ctx).Where("id = ?", id).First(&bookChapter).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bookChapter, nil
}

func (b bookChapterRepository) GetByBookIDAndOrder(ctx context.Context, id string, order int) (*model.BookChapter, error) {
	var bookChapter model.BookChapter
	err := b.db.WithContext(ctx).Where("book_id = ? AND order = ?", id, order).First(&bookChapter).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bookChapter, nil
}

func (b bookChapterRepository) GetIDListByBookID(ctx context.Context, id string) (*[]model.BookChapterInfo, error) {
	var bookChapterInfo *[]model.BookChapterInfo
	err := b.db.WithContext(ctx).Where("book_id = ?", id).Order("`order` ASC").Find(&bookChapterInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return bookChapterInfo, nil
}

func (b bookChapterRepository) AddByBookID(ctx context.Context, chapter *model.BookChapter) (*model.BookChapter, error) {
	var bookChapterInfo model.BookChapterInfo
	err := b.db.WithContext(ctx).Model(&model.BookChapter{}).Where("book_id = ?", chapter.BookID).Order("`order` DESC").First(&bookChapterInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			bookChapterInfo.Order = 0
		} else {
			return nil, err
		}
	}
	bookChapter := model.BookChapter{
		ID:       uuid.New().String(),
		BookID:   chapter.BookID,
		Title:    chapter.Title,
		Content:  chapter.Content,
		Order:    bookChapterInfo.Order + 1,
		VIPLevel: chapter.VIPLevel,
		Price:    chapter.Price,
	}
	err = b.db.WithContext(ctx).Create(&bookChapter).Error
	if err != nil {
		return nil, err
	}
	return &bookChapter, nil
}

func (b bookChapterRepository) CountByBookID(ctx context.Context, bookID string) (int, error) {
	var count int64
	err := b.db.WithContext(ctx).Where("book_id", bookID).Count(&count).Error
	return int(count), err
}

func (b bookChapterRepository) Delete(ctx context.Context, id string) error {
	return b.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&model.BookChapter{}).Error
}

func (b bookChapterRepository) Update(ctx context.Context, id string, bookChapter *model.BookChapter) error {
	return b.db.WithContext(ctx).Where("id = ?", id).Save(bookChapter).Error
}
