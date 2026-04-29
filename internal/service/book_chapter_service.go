package service

import (
	"context"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/utils"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrBookChapterNotFound = errors.New("book_chapter not found")
	ErrVIPLevel            = errors.New("the membership level is insufficient")
	ErrPermission          = errors.New("user permission denied")
)

type BookChapterService interface {
	AddByBookID(ctx context.Context, userID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterResponse, error)
	GuestGetChapter(ctx context.Context, bookID string) (*dto.BookChapterResponse, error)
	GetChapter(ctx context.Context, userID string, bookChapterID string) (*dto.BookChapterResponse, error)
	UpdateChapter(ctx context.Context, userID string, req *dto.UpdateBookChapterRequest) (*dto.BookChapterResponse, error)
	DeleteChapter(ctx context.Context, userID string, chapterID string) error
	GetChapterList(ctx context.Context, req *dto.GetBookChapterListRequest) (*dto.GetBookChapterListResponse, error)
	BuyBookChapter(ctx context.Context, userID string, req *dto.BuyBookChapterRequest) error
	GetChapterInfo(ctx context.Context, bookChapterID string) (*dto.BookChapterInfoResponse, error)
}

type bookChapterService struct {
	userRepo                 repository.UserRepository
	bookRepo                 repository.BookRepository
	bookChapterRepo          repository.BookChapterRepository
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository
	userAssetRepo            repository.UserAssetRepository
	bookCache                cache.BookCache
	bookChapterCache         cache.BookChapterCache
	userStatCache            cache.UserStatCache
	limitCache               cache.LimitCache
	db                       *gorm.DB
	cfg                      *config.Config
}

func NewBookChapterService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	bookChapterRepo repository.BookChapterRepository,
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository,
	userAssetRepo repository.UserAssetRepository,
	bookCache cache.BookCache,
	bookChapterCache cache.BookChapterCache,
	userStatCache cache.UserStatCache,
	limitCache cache.LimitCache,
	db *gorm.DB,
	cfg *config.Config,
) BookChapterService {
	return &bookChapterService{
		userRepo:                 userRepo,
		bookRepo:                 bookRepo,
		bookChapterRepo:          bookChapterRepo,
		userPurchasedChapterRepo: userPurchasedChapterRepo,
		userAssetRepo:            userAssetRepo,
		bookCache:                bookCache,
		bookChapterCache:         bookChapterCache,
		userStatCache:            userStatCache,
		limitCache:               limitCache,
		db:                       db,
		cfg:                      cfg,
	}
}

func (s bookChapterService) AddByBookID(ctx context.Context, userID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterResponse, error) {
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

	bookChapter := &model.BookChapter{
		ID:       uuid.New().String(),
		BookID:   req.BookID,
		Title:    *req.Title,
		Content:  *req.Content,
		VIPLevel: VIPLevelToStr(VIPStrToLevel(req.VIPLevel)),
		Price:    int64(req.Price),
	}

	bookChapterResp, err := s.bookChapterRepo.AddByBookID(ctx, bookChapter)
	if err != nil {
		return nil, err
	}

	return toBookChapterResponse(s.cfg, bookChapterResp, book), nil

}

func toBookChapterInfoResponse(cfg *config.Config, info *model.BookChapterInfo, book *model.Book) *dto.BookChapterInfoResponse {
	return &dto.BookChapterInfoResponse{
		ChapterID: info.ID,
		Title:     info.Title,
		Order:     info.Order,
		VIPLevel:  info.VIPLevel,
		Price:     info.Price,
		Book:      toBookResponse(cfg, book),
	}
}

func toBookChapterResponse(cfg *config.Config, info *model.BookChapter, book *model.Book) *dto.BookChapterResponse {
	return &dto.BookChapterResponse{
		ChapterID: info.ID,
		Title:     info.Title,
		Content:   info.Content,
		Order:     info.Order,
		VIPLevel:  info.VIPLevel,
		Price:     info.Price,
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

	if VIPStrToLevel(bookChapter.VIPLevel) > VIPStrToLevel("") {
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

	// 先检查是否购买
	existBuy, err := s.userPurchasedChapterRepo.GetByChapterIDAndUserID(ctx, bookChapterID, userID)
	if err != nil {
		return nil, err
	}
	if existBuy == nil {
		// 没有购买书籍则判断vip是否等级足够
		userAsset, err := s.userAssetRepo.GetByID(ctx, s.db, userID)
		if err != nil {
			return nil, err
		}
		if VIPStrToLevel(bookChapter.VIPLevel) > VIPStrToLevel(userAsset.VIPLevel) {
			return nil, ErrVIPLevel
		}
	}

	// 向redis里面添加热力值
	err = func() error {
		// 用户添加对应书籍数量
		_, _ = s.userStatCache.AddReadBook(ctx, userID, book.ID)

		// 阅读频次需要大于1min
		exist, _ := s.limitCache.CheckUserBrowse1Min(ctx, userID)
		if exist {
			return nil
		}

		// 用户添加对应阅读量
		pageWords := int64(utils.CountPassageWord(bookChapter.Content))
		addReadWordsErr := s.userStatCache.IncreaseReadWordsNum(ctx, userID, pageWords)
		if addReadWordsErr != nil {
			logger.Fatal(fmt.Sprintf("increase read words fail: %v", err))
		}

		// 每阅读5min记录到书籍
		exist, _ = s.limitCache.CheckUserBrowseBook5Min(ctx, userID, book.ID)
		if exist {
			return nil
		}
		// 热力值限流
		_ = s.limitCache.AddUserBrowseBook5Min(ctx, userID, book.ID)

		_ = s.limitCache.AddUserBrowse1Min(ctx, userID)

		_ = s.bookCache.IncreaseBookThermalDaily(ctx, book.ID, 1)

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

func (s bookChapterService) UpdateChapter(ctx context.Context, userID string, req *dto.UpdateBookChapterRequest) (*dto.BookChapterResponse, error) {
	old, err := s.bookChapterRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, ErrBookChapterNotFound
	}

	book, err := s.bookRepo.GetByID(ctx, old.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	if book.OwnerID != userID {
		return nil, ErrPermission
	}

	cur := old

	if req.Title != nil && *req.Title != "" {
		cur.Title = *req.Title
	}
	if req.Content != nil && *req.Content != "" {
		cur.Content = *req.Content
	}
	if req.VIPLevel != nil {
		cur.VIPLevel = VIPLevelToStr(VIPStrToLevel(*req.VIPLevel))
	}
	if req.Price != nil {
		cur.Price = *req.Price
	}

	err = s.bookChapterRepo.Update(ctx, old.ID, cur)
	if err != nil {
		return nil, err
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
				VIPLevel:  chapterInfo.VIPLevel,
				Price:     chapterInfo.Price,
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

func (s bookChapterService) BuyBookChapter(ctx context.Context, userID string, req *dto.BuyBookChapterRequest) error {
	bookChapter, err := s.bookChapterRepo.GetByID(ctx, req.ChapterID)
	if err != nil {
		return err
	}
	if bookChapter == nil {
		return ErrBookNotFound
	}

	if bookChapter.Price != req.Price {
		return ErrParam
	}
	if bookChapter.BookID != req.BookID {
		return ErrParam
	}

	tx := s.db.Begin()
	exist, err := s.userPurchasedChapterRepo.GetByChapterIDAndUserID(ctx, req.ChapterID, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if exist != nil {
		tx.Rollback()
		return ErrBookAlreadyBuy
	}

	purchasedChapter := &model.UserPurchasedChapter{
		ID:        uuid.New().String(),
		UserID:    userID,
		ChapterID: req.ChapterID,
		BookID:    req.BookID,
		ActiveAt:  time.Now().Add(3 * 365 * 24 * time.Hour),
	}
	userAsset, err := s.userAssetRepo.GetByID(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if userAsset == nil {
		tx.Rollback()
		return ErrUserNotFound
	}
	if userAsset.Gold < req.Price {
		tx.Rollback()
		return ErrSufficientFunds
	}

	_, err = s.userAssetRepo.SubGoldByID(ctx, tx, userID, req.Price)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = s.userPurchasedChapterRepo.AddPurchasedChapter(ctx, purchasedChapter)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	return nil
}

func (s bookChapterService) GetChapterInfo(ctx context.Context, bookChapterID string) (*dto.BookChapterInfoResponse, error) {
	bookChapter, exist, _ := s.bookChapterCache.GetBookChapter(ctx, bookChapterID)
	var err error
	if !exist {
		bookChapter, err = s.bookChapterRepo.GetByID(ctx, bookChapterID)
		if err != nil {
			return nil, err
		}
		if bookChapter == nil {
			return nil, ErrBookChapterNotFound
		}
		_ = s.bookChapterCache.UpdateBookChapter(ctx, bookChapter)
	}

	book, exist, _ := s.bookCache.GetBook(ctx, bookChapter.BookID)
	if !exist {
		book, err = s.bookRepo.GetByID(ctx, bookChapter.BookID)
		if err != nil {
			return nil, err
		}
		if book == nil {
			return nil, ErrBookNotFound
		}
		_ = s.bookCache.UpdateBook(ctx, book)
	}

	return &dto.BookChapterInfoResponse{
		ChapterID: bookChapter.ID,
		Title:     bookChapter.Title,
		Order:     bookChapter.Order,
		VIPLevel:  bookChapter.VIPLevel,
		Price:     bookChapter.Price,
		Book:      toBookResponse(s.cfg, book),
	}, nil
}
