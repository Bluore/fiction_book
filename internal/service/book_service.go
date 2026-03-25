package service

import (
	"context"
	"errors"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"github.com/google/uuid"
)

var (
	ErrBookExists = errors.New("book already exists")
)

// BookService 用户服务接口
type BookService interface {
	Create(ctx context.Context, req *dto.CreateBookRequest) (*dto.CreateBookResponse, error)
	//GetByID(ctx context.Context, id string) (*dto., error)
	//List(ctx context.Context, page int, size int) ([]*model.Book, error)
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

func (s *bookService) Create(ctx context.Context, req *dto.CreateBookRequest) (*dto.CreateBookResponse, error) {
	existingBook, err := s.bookRepo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if existingBook == nil {
		return nil, ErrBookExists
	}

	book := &model.Book{
		ID:          uuid.New().String(),
		OwnerID:     req.OwnerId,
		Name:        req.Name,
		Description: req.Description,
		CoverImage:  req.CoverImage,
	}
	err = s.bookRepo.Create(ctx, book)
	if err != nil {
		return nil, err
	}
	return s.toBookResponse(book), nil
}

func (s *bookService) toBookResponse(book *model.Book) *dto.CreateBookResponse {
	return &dto.CreateBookResponse{
		Name:        book.Name,
		Description: book.Description,
		CoverImage:  book.CoverImage,
		OwnerID:     book.OwnerID,
		CreatedAt:   book.CreatedAt,
		UpdatedAt:   book.UpdatedAt,
	}
}
