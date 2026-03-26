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
	ErrBookExists   = errors.New("book already exists")
	ErrBookNotFound = errors.New("book not found")
)

// BookService 用户服务接口
type BookService interface {
	Create(ctx context.Context, req *dto.CreateBookRequest) (*dto.BookResponse, error)
	GetByID(ctx context.Context, id string) (*dto.BookResponse, error)
	List(ctx context.Context, page int, size int) (*dto.BookListResponse, error)
	ListBooks(ctx context.Context, req *dto.ListBooksRequest) (*dto.BookListResponse, error)
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

func (s *bookService) Create(ctx context.Context, req *dto.CreateBookRequest) (*dto.BookResponse, error) {
	existingBook, err := s.bookRepo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if existingBook != nil {
		return nil, ErrBookExists
	}

	book := &model.Book{
		ID:              uuid.New().String(),
		OwnerID:         req.OwnerId,
		Name:            req.Name,
		Description:     req.Description,
		VIPLevel:        req.VIPLevel,
		Price:           req.Price,
		CoverImage:      req.CoverImage,
		PushedAt:        req.PushAt,
		BrowseAmount:    0,
		BookshelfAmount: 0,
	}
	err = s.bookRepo.Create(ctx, book)
	if err != nil {
		return nil, err
	}
	return s.toBookResponse(book), nil
}

func (s *bookService) toBookResponse(book *model.Book) *dto.BookResponse {
	return &dto.BookResponse{
		ID:              book.ID,
		Name:            book.Name,
		Description:     book.Description,
		CoverImage:      book.CoverImage,
		OwnerID:         book.OwnerID,
		VIPLevel:        book.VIPLevel,
		Price:           book.Price,
		PushAt:          book.PushedAt,
		BrowseAmount:    book.BrowseAmount,
		BookshelfAmount: book.BookshelfAmount,
		CreatedAt:       book.CreatedAt,
		UpdatedAt:       book.UpdatedAt,
	}
}

func (s *bookService) GetByID(ctx context.Context, id string) (*dto.BookResponse, error) {
	book, err := s.bookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}
	return s.toBookResponse(book), nil
}

func (s *bookService) List(ctx context.Context, page int, size int) (*dto.BookListResponse, error) {
	books, err := s.bookRepo.List(ctx, page, size)
	if err != nil {
		return nil, err
	}

	count, err := s.bookRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	var bookResponses []*dto.BookResponse
	for _, book := range books {
		bookResponses = append(bookResponses, s.toBookResponse(book))
	}

	pageCount := &dto.PageListResponse{
		Page:  page,
		Size:  size,
		Count: count,
	}

	bookListResponse := &dto.BookListResponse{
		Books:            bookResponses,
		PageListResponse: *pageCount,
	}
	return bookListResponse, nil
}

func (s *bookService) ListBooks(ctx context.Context, req *dto.ListBooksRequest) (*dto.BookListResponse, error) {
	req.ToStandard()
	books, err := s.bookRepo.ListWithOption(ctx, repository.ListOption{
		Page:      req.Page,
		Size:      req.Size,
		SortKey:   req.SortKey,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return nil, err
	}

	count, err := s.bookRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	var bookResponses []*dto.BookResponse
	for _, book := range books {
		bookResponses = append(bookResponses, s.toBookResponse(book))
	}

	return &dto.BookListResponse{
		Books: bookResponses,
		PageListResponse: dto.PageListResponse{
			Page:  req.Page,
			Size:  req.Size,
			Count: count,
		},
	}, nil
}
