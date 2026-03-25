package repository

import (
	"gorm.io/gorm"
)

// UserRepository 用户仓储接口
type BookRepository interface {
}

type bookRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例
func NewBookRepository(db *gorm.DB) BookRepository {
	return &bookRepository{db: db}
}
