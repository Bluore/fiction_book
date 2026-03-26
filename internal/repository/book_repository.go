package repository

import (
	"context"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"gorm.io/gorm"
)

// ListOption 通用列表查询选项
type ListOption struct {
	Page      int
	Size      int
	SortKey   dto.BookSortKey
	SortOrder dto.SortOrder
}

// MapSortKeyToColumn 将 SortKey 映射为 GORM Order 字符串
func MapSortKeyToColumn(sortKey dto.BookSortKey, order dto.SortOrder) string {
	columnMap := map[dto.BookSortKey]string{
		dto.SortByBrowseAmount:    "browse_amount",
		dto.SortByBookshelfAmount: "bookshelf_amount",
	}
	col, ok := columnMap[sortKey]
	if !ok {
		return "browse_amount"
	}
	return col + " " + string(order)
}

// BookRepository 用户仓储接口
type BookRepository interface {
	GetByID(ctx context.Context, id string) (*model.Book, error)
	GetByOwnerID(ctx context.Context, ownerID string) ([]*model.Book, error)
	GetByName(ctx context.Context, name string) (*model.Book, error)
	List(ctx context.Context, page int, size int) ([]*model.Book, error)
	ListWithOption(ctx context.Context, opt ListOption) ([]*model.Book, error)
	Create(ctx context.Context, model *model.Book) error
	Update(ctx context.Context, model *model.Book) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
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
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &book, nil
}

func (r *bookRepository) GetByOwnerID(ctx context.Context, ownerID string) ([]*model.Book, error) {
	var books []*model.Book
	err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Find(&books).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return books, nil
}

func (r *bookRepository) GetByName(ctx context.Context, name string) (*model.Book, error) {
	var book model.Book
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&book).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &book, nil
}

func (r *bookRepository) List(ctx context.Context, page int, size int) ([]*model.Book, error) {
	var books []*model.Book
	err := r.db.WithContext(ctx).Limit(size).Offset((page - 1) * size).Find(&books).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return books, nil
}

func (r *bookRepository) ListWithOption(ctx context.Context, opt ListOption) ([]*model.Book, error) {
	var books []*model.Book
	orderClause := MapSortKeyToColumn(opt.SortKey, opt.SortOrder)
	err := r.db.WithContext(ctx).
		Order(orderClause).
		Limit(opt.Size).
		Offset((opt.Page - 1) * opt.Size).
		Find(&books).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
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

func (r *bookRepository) Count(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Book{}).Count(&count).Error
	return int(count), err
}
