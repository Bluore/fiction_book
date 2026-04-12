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
	ErrVIPLevel            = errors.New("the membership level is insufficient")
	ErrPermission          = errors.New("user permission denied")
)

type BookChapterService interface {
	AddByBookID(ctx context.Context, userID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterInfoResponse, error)
	GuestGetChapter(ctx context.Context, bookID string) (*dto.BookChapterResponse, error)
	GetChapter(ctx context.Context, userID string, bookID string) (*dto.BookChapterResponse, error)
	UpdateChapter(ctx context.Context, req *dto.UpdateBookChapterRequest) (*dto.BookChapterResponse, error)
}

type bookChapterService struct {
	userRepo        repository.UserRepository
	bookRepo        repository.BookRepository
	bookChapterRepo repository.BookChapterRepository
	cfg             *config.Config
}

func NewBookChapterService(userRepo repository.UserRepository, bookRepo repository.BookRepository, bookChapterRepo repository.BookChapterRepository, cfg *config.Config) BookChapterService {
	return &bookChapterService{
		userRepo:        userRepo,
		bookRepo:        bookRepo,
		bookChapterRepo: bookChapterRepo,
		cfg:             cfg,
	}
}

func (b bookChapterService) AddByBookID(ctx context.Context, userID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterInfoResponse, error) {
	book, err := b.bookRepo.GetByID(ctx, req.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	if book.OwnerID != userID {
		return nil, ErrPermission
	}

	bookInfo, err := b.bookChapterRepo.AddByBookID(ctx, req.BookID, *req.Title, *req.Content)
	if err != nil {
		return nil, err
	}
	return toBookChapterInfoResponse(b.cfg, bookInfo, book), nil

}

func toBookChapterInfoResponse(cfg *config.Config, info *model.BookChapterInfo, book *model.Book) *dto.BookChapterInfoResponse {
	return &dto.BookChapterInfoResponse{
		ChapterID: info.ID,
		Title:     info.Title,
		Order:     info.Order,
		Book:      toBookResponse(cfg, book),
	}
}

func toBookChapterResponse(cfg *config.Config, info *model.BookChapter, book *model.Book) *dto.BookChapterResponse {
	return &dto.BookChapterResponse{
		ChapterID: info.ID,
		Title:     info.Title,
		Content:   info.Content,
		Order:     info.Order,
		Book:      toBookResponse(cfg, book),
	}
}

func (b bookChapterService) GuestGetChapter(ctx context.Context, bookID string) (*dto.BookChapterResponse, error) {
	bookChapter, err := b.bookChapterRepo.GetByID(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if bookChapter == nil {
		return nil, ErrBookChapterNotFound
	}

	book, err := b.bookRepo.GetByID(ctx, bookChapter.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	if book.VIPLevel != "" {
		return nil, ErrUserNotLogin
	}
	return toBookChapterResponse(b.cfg, bookChapter, book), nil
}

func (b bookChapterService) GetChapter(ctx context.Context, userID string, bookID string) (*dto.BookChapterResponse, error) {
	user, err := b.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	bookChapter, err := b.bookChapterRepo.GetByID(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if bookChapter == nil {
		return nil, ErrBookChapterNotFound
	}

	book, err := b.bookRepo.GetByID(ctx, bookChapter.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	// todo 先检查是否购买

	// todo vip获取和比较
	if VIPStrToLevle(book.VIPLevel) > VIPStrToLevle("vip_0") {
		return nil, ErrVIPLevel
	}

	return toBookChapterResponse(b.cfg, bookChapter, book), nil
}

func VIPStrToLevle(vipStr string) int {
	switch vipStr {
	case "":
		return -1
	case "vip_0":
		return 0
	case "vip_1":
		return 1
	case "vip_2":
		return 2
	case "vip_3":
		return 3
	case "off":
		return 999
	default:
		return 0
	}
}

func VIPLevelToStr(vipLevel int) string {
	switch vipLevel {
	case -1:
		return ""
	case 0:
		return "vip_0"
	case 1:
		return "vip_1"
	case 2:
		return "vip_2"
	case 3:
		return "vip_3"
	default:
		return "off"
	}
}

func (b bookChapterService) UpdateChapter(ctx context.Context, req *dto.UpdateBookChapterRequest) (*dto.BookChapterResponse, error) {
	old, err := b.bookChapterRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, ErrBookChapterNotFound
	}
	cur := old

	if req.Title != nil && *req.Title != "" {
		cur.Title = *req.Title
	}
	if req.Content != nil && *req.Content != "" {
		cur.Content = *req.Content
	}

	err = b.bookChapterRepo.Update(ctx, old.ID, cur)
	if err != nil {
		return nil, err
	}

	book, err := b.bookRepo.GetByID(ctx, old.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	return toBookChapterResponse(b.cfg, cur, book), nil

}
