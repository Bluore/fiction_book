package service

import (
	"context"
	"errors"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
)

var (
	ErrBookChapterNotFound = errors.New("book_chapter not found")
)

type BookChapterService interface {
	AddByBookID(ctx context.Context, bookID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterInfoResponse, error)
}

type bookChapterService struct {
	bookRepo        repository.BookRepository
	bookChapterRepo repository.BookChapterRepository
	cfg             *config.Config
}

func NewBookChapterService(bookRepo repository.BookRepository, bookChapterRepo repository.BookChapterRepository, cfg *config.Config) BookChapterService {
	return &bookChapterService{
		bookRepo:        bookRepo,
		bookChapterRepo: bookChapterRepo,
		cfg:             cfg,
	}
}

func (b bookChapterService) AddByBookID(ctx context.Context, bookID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterInfoResponse, error) {
	book, err := b.bookRepo.GetByID(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	bookInfo, err := b.bookChapterRepo.AddByBookID(ctx, bookID, *req.Title, *req.Content)
	if err != nil {
		return nil, err
	}
	return toBookChapterInfoResponse(bookInfo, book), nil

}

func toBookChapterInfoResponse(info *model.BookChapterInfo, book *model.Book) *dto.BookChapterInfoResponse {
	return &dto.BookChapterInfoResponse{
		ChapterID: info.ID,
		Title:     info.Title,
		Order:     info.Order,
		Book:      toBookResponse(book),
	}
}
