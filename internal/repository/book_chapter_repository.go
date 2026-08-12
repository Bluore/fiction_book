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

	BookChapterInfoGetByID(ctx context.Context, db *gorm.DB, id string) (*model.BookChapterInfo, error)

	BookChapterContentGetByID(ctx context.Context, db *gorm.DB, id string) (*model.BookChapterContent, error)
}

type bookChapterRepository struct {
	db *gorm.DB
}

func NewBookChapterRepository(db *gorm.DB) BookChapterRepository {
	return &bookChapterRepository{db: db}
}

func (r bookChapterRepository) GetByID(ctx context.Context, id string) (*model.BookChapter, error) {
	var bookChapter model.BookChapter
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&bookChapter).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bookChapter, nil
}

func (r bookChapterRepository) GetByBookIDAndOrder(ctx context.Context, id string, order int) (*model.BookChapter, error) {
	var bookChapter model.BookChapter
	err := r.db.WithContext(ctx).Where("book_id = ? AND order = ?", id, order).First(&bookChapter).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bookChapter, nil
}

func (r bookChapterRepository) GetIDListByBookID(ctx context.Context, id string) (*[]model.BookChapterInfo, error) {
	var bookChapterInfo *[]model.BookChapterInfo
	err := r.db.WithContext(ctx).Where("book_id = ?", id).Order("`order` ASC").Find(&bookChapterInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return bookChapterInfo, nil
}

func (r bookChapterRepository) AddByBookID(ctx context.Context, chapter *model.BookChapter) (*model.BookChapter, error) {
	var bookChapterInfo model.BookChapterInfo
	err := r.db.WithContext(ctx).Model(&model.BookChapter{}).Where("book_id = ?", chapter.BookID).Order("`order` DESC").First(&bookChapterInfo).Error
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
	err = r.db.WithContext(ctx).Create(&bookChapter).Error
	if err != nil {
		return nil, err
	}
	return &bookChapter, nil
}

func (r bookChapterRepository) CountByBookID(ctx context.Context, bookID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Where("book_id", bookID).Count(&count).Error
	return int(count), err
}

func (r bookChapterRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&model.BookChapter{}).Error
}

func (r bookChapterRepository) Update(ctx context.Context, id string, bookChapter *model.BookChapter) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Save(bookChapter).Error
}

func (r bookChapterRepository) BookChapterInfoGetByID(ctx context.Context, db *gorm.DB, id string) (*model.BookChapterInfo, error) {
	var bookChapterInfo model.BookChapterInfo
	err := db.WithContext(ctx).Where("id = ?", id).First(&bookChapterInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bookChapterInfo, nil
}

func (r bookChapterRepository) BookChapterContentGetByID(ctx context.Context, db *gorm.DB, id string) (*model.BookChapterContent, error) {
	var bookChapterContent model.BookChapterContent
	err := db.WithContext(ctx).Where("id = ?", id).First(&bookChapterContent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bookChapterContent, nil
}
