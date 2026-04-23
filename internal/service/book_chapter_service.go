package service

import (
	"context"
	"errors"
	"fiction-book/internal/catch"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrBookChapterNotFound = errors.New("book_chapter not found")
	ErrVIPLevel            = errors.New("the membership level is insufficient")
	ErrPermission          = errors.New("user permission denied")
)

type BookChapterService interface {
	AddByBookID(ctx context.Context, userID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterInfoResponse, error)
	GuestGetChapter(ctx context.Context, bookID string) (*dto.BookChapterResponse, error)
	GetChapter(ctx context.Context, userID string, bookChapterID string) (*dto.BookChapterResponse, error)
	UpdateChapter(ctx context.Context, req *dto.UpdateBookChapterRequest) (*dto.BookChapterResponse, error)
	DeleteChapter(ctx context.Context, userID string, chapterID string) error
	GetChapterList(ctx context.Context, req *dto.GetBookChapterListRequest) (*dto.GetBookChapterListResponse, error)
}

type bookChapterService struct {
	userRepo         repository.UserRepository
	bookRepo         repository.BookRepository
	bookChapterRepo  repository.BookChapterRepository
	bookCatch        catch.BookCatch
	bookChapterCatch catch.BookChapterCatch
	limitCatch       catch.LimitCatch
	db               *gorm.DB
	cfg              *config.Config
}

func NewBookChapterService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	bookChapterRepo repository.BookChapterRepository,
	bookCatch catch.BookCatch,
	bookChapterCatch catch.BookChapterCatch,
	limitCatch catch.LimitCatch,
	db *gorm.DB,
	cfg *config.Config,
) BookChapterService {
	return &bookChapterService{
		userRepo:         userRepo,
		bookRepo:         bookRepo,
		bookChapterRepo:  bookChapterRepo,
		bookCatch:        bookCatch,
		bookChapterCatch: bookChapterCatch,
		limitCatch:       limitCatch,
		db:               db,
		cfg:              cfg,
	}
}

func (s bookChapterService) AddByBookID(ctx context.Context, userID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterInfoResponse, error) {
	book, err := s.bookRepo.GetByID(ctx, req.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	if book.OwnerID != userID {
		return nil, ErrPermission
	}

	bookInfo, err := s.bookChapterRepo.AddByBookID(ctx, req.BookID, *req.Title, *req.Content)
	if err != nil {
		return nil, err
	}
	return toBookChapterInfoResponse(s.cfg, bookInfo, book), nil

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

func (s bookChapterService) GuestGetChapter(ctx context.Context, bookID string) (*dto.BookChapterResponse, error) {
	bookChapter, err := s.bookChapterRepo.GetByID(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if bookChapter == nil {
		return nil, ErrBookChapterNotFound
	}

	book, err := s.bookRepo.GetByID(ctx, bookChapter.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	if book.VIPLevel != "" {
		return nil, ErrUserNotLogin
	}
	return toBookChapterResponse(s.cfg, bookChapter, book), nil
}

func (s bookChapterService) GetChapter(ctx context.Context, userID string, bookChapterID string) (*dto.BookChapterResponse, error) {
	user, err := s.userRepo.GetByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	bookChapter, err := s.bookChapterRepo.GetByID(ctx, bookChapterID)
	if err != nil {
		return nil, err
	}
	if bookChapter == nil {
		return nil, ErrBookChapterNotFound
	}

	book, err := s.bookRepo.GetByID(ctx, bookChapter.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	// todo 先检查是否购买

	// vip获取和比较
	if VIPStrToLevel(book.VIPLevel) > VIPStrToLevel("vip_0") {
		return nil, ErrVIPLevel
	}

	// 向redis里面添加热力值
	err = func() error {
		exist, _ := s.limitCatch.CheckUserBrowseBook5Min(ctx, userID, book.ID)
		if exist {
			return nil
		}
		exist, _ = s.limitCatch.CheckUserBrowse1Min(ctx, userID)
		if exist {
			return nil
		}
		// 热力值限流
		_ = s.limitCatch.AddUserBrowseBook5Min(ctx, userID, book.ID)

		_ = s.limitCatch.AddUserBrowse1Min(ctx, userID)

		_ = s.bookCatch.IncreaseBookThermalDailyCatch(ctx, book.ID, 10)

		// 数据库添加数据
		_, err = s.bookRepo.IncreaseOfBrowseAmount(ctx, s.db, book.ID)
		if err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		logger.Error(fmt.Sprintf("addThermalDaily error: %v", err))
	}

	return toBookChapterResponse(s.cfg, bookChapter, book), nil
}

func VIPStrToLevel(vipStr string) int {
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

func (s bookChapterService) UpdateChapter(ctx context.Context, req *dto.UpdateBookChapterRequest) (*dto.BookChapterResponse, error) {
	old, err := s.bookChapterRepo.GetByID(ctx, req.ID)
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

	err = s.bookChapterRepo.Update(ctx, old.ID, cur)
	if err != nil {
		return nil, err
	}

	book, err := s.bookRepo.GetByID(ctx, old.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	return toBookChapterResponse(s.cfg, cur, book), nil

}

func (s bookChapterService) DeleteChapter(ctx context.Context, userID string, chapterID string) error {
	// 检查章节是否存在
	chapter, err := s.bookChapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return ErrBookChapterNotFound
	}

	// 检查书籍是否存在
	book, err := s.bookRepo.GetByID(ctx, chapter.BookID)
	if err != nil {
		return err
	}
	if book == nil {
		return ErrBookNotFound
	}

	// 删除章节
	return s.bookChapterRepo.Delete(ctx, chapterID)
}

func (s bookChapterService) GetChapterList(ctx context.Context, req *dto.GetBookChapterListRequest) (*dto.GetBookChapterListResponse, error) {
	exist, err := s.bookRepo.GetByID(ctx, req.BookID)
	if err != nil {
		return nil, err
	}
	if exist == nil {
		return nil, ErrBookNotFound
	}

	chapterList, err := s.bookChapterRepo.GetIDListByBookID(ctx, req.BookID)
	if err != nil {
		return nil, err
	}

	var chapterListShow []dto.BookChapterCatalogueResponse
	for _, chapterInfo := range *chapterList {
		chapterListShow = append(
			chapterListShow,
			dto.BookChapterCatalogueResponse{
				ChapterID: chapterInfo.ID,
				Title:     chapterInfo.Title,
				Order:     chapterInfo.Order,
			},
		)
	}

	if len(chapterListShow) == 0 {
		chapterListShow = make([]dto.BookChapterCatalogueResponse, 0)
	}

	resp := &dto.GetBookChapterListResponse{
		Chapters: chapterListShow,
	}

	return resp, nil
}
