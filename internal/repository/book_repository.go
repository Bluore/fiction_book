package repository

import (
	"context"
	"fiction-book/internal/model"
	"gorm.io/gorm"
)

// BookRepository 用户仓储接口
type BookRepository interface {
	GetByID(ctx context.Context, id string) (*model.Book, error)
	GetByUsername(ctx context.Context, username string) ([]*model.Book, error)
	List(ctx context.Context) ([]*model.Book, error)
	Create(ctx context.Context, model *model.Book) error
	Update(ctx context.Context, model *model.Book) error
	Delete(ctx context.Context, id string) error
}

type bookRepository struct {
	db *gorm.DB
}

// NewBookRepository 创建用户仓储实例
func NewBookRepository(db *gorm.DB) BookRepository {
	return &bookRepository{db: db}
}

func (r *bookRepository) GetByID(ctx context.Context, id string) (*model.Book, error) {
	var book model.Book
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&book).Error
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *bookRepository) GetByUsername(ctx context.Context, username string) ([]*model.Book, error) {
	var books []*model.Book
	err := r.db.WithContext(ctx).Where("username = ?", username).Find(&books).Error
	if err != nil {
		return nil, err
	}
	return books, nil
}

func (r *bookRepository) List(ctx context.Context) ([]*model.Book, error) {
	var books []*model.Book
	err := r.db.WithContext(ctx).Find(&books).Error
	if err != nil {
		return nil, err
	}
	return books, nil
}

func (r *bookRepository) Create(ctx context.Context, model *model.Book) error {
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *bookRepository) Update(ctx context.Context, model *model.Book) error {
	return r.db.WithContext(ctx).Save(&model).Error
}

func (r *bookRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Book{}).Error
}
