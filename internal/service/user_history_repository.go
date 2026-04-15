package service

import (
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	"gorm.io/gorm"
)

type UserHistoryService interface {
}

type userHistoryService struct {
	userRepo        repository.UserRepository
	userHistoryRepo repository.UserHistoryRepository
	bookRepo        repository.BookRepository
	db              *gorm.DB
	cfg             *config.Config
}

// NewUserHistoryService 创建用户访问书籍历史的业务层
func NewUserHistoryService(
	userRepo repository.UserRepository,
	userHistoryRepo repository.UserHistoryRepository,
	bookRepo repository.BookRepository,
	db *gorm.DB,
	cfg *config.Config,
) UserHistoryService {
	return &userHistoryService{
		userRepo:        userRepo,
		userHistoryRepo: userHistoryRepo,
		bookRepo:        bookRepo,
		db:              db,
		cfg:             cfg,
	}
}
