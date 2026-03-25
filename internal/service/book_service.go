package service

import (
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
)

// BookService 用户服务接口
type BookService interface {
}

type bookService struct {
	bookRepo repository.BookRepository
	cfg      *config.Config
}

// NewBookService 创建用户服务实例
func NewBookService(bookRepo repository.BookRepository, cfg *config.Config) BookService {
	return &bookService{
		bookRepo: bookRepo,
		cfg:      cfg,
	}
}
