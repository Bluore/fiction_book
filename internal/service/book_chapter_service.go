package service

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/utils"
	"fmt"
	"strings"
	"time"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
	SearchBookChaptersV2(ctx context.Context, req *dto.SearchBookChaptersV2Request) (*dto.SearchBookChaptersV2Response, error)
}

type bookChapterService struct {
	userRepo                 repository.UserRepository
	bookRepo                 repository.BookRepository
	bookChapterRepo          repository.BookChapterRepository
	bookChapterESRepo        repository.BookChapterESRepository
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository
	userAssetRepo            repository.UserAssetRepository
	userBookshelfRepo        repository.UserBookshelfRepository
	feedRepo                 repository.FeedRepository
	feedCursorRepo           repository.FeedCursorRepository
	outboxRepo               repository.OutboxRepository
	bookCache                cache.BookCache
	bookThermalCache         cache.BookThermalCache
	bookChapterCache         cache.BookChapterCache
	userStatCache            cache.UserStatCache
	limitCache               cache.LimitCache
	feedCache                cache.FeedCache
	feedClient               kafka_client.FeedClient
	db                       *gorm.DB
	cfg                      *config.Config
	pool                     *agilePool.Pool
}

func NewBookChapterService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	bookChapterRepo repository.BookChapterRepository,
	bookChapterESRepo repository.BookChapterESRepository,
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository,
	userAssetRepo repository.UserAssetRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	feedRepo repository.FeedRepository,
	feedCursorRepo repository.FeedCursorRepository,
	outboxRepo repository.OutboxRepository,
	bookCache cache.BookCache,
	bookThermalCache cache.BookThermalCache,
	bookChapterCache cache.BookChapterCache,
	userStatCache cache.UserStatCache,
	limitCache cache.LimitCache,
	feedCache cache.FeedCache,
	feedClient kafka_client.FeedClient,
	db *gorm.DB,
	cfg *config.Config,
	pool *agilePool.Pool,
) BookChapterService {
	return &bookChapterService{
		userRepo:                 userRepo,
		bookRepo:                 bookRepo,
		bookChapterRepo:          bookChapterRepo,
		bookChapterESRepo:        bookChapterESRepo,
		userPurchasedChapterRepo: userPurchasedChapterRepo,
		userAssetRepo:            userAssetRepo,
		userBookshelfRepo:        userBookshelfRepo,
		feedRepo:                 feedRepo,
		feedCursorRepo:           feedCursorRepo,
		outboxRepo:               outboxRepo,
		bookCache:                bookCache,
		bookThermalCache:         bookThermalCache,
		bookChapterCache:         bookChapterCache,
		userStatCache:            userStatCache,
		limitCache:               limitCache,
		feedCache:                feedCache,
		feedClient:               feedClient,
		db:                       db,
		cfg:                      cfg,
		pool:                     pool,
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
		VIPLevel: VIPLevelToStr(VIPStrToLevel(*req.VIPLevel)),
		Price:    int64(*req.Price),
	}

	var bookChapterResp *model.BookChapter
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		bookChapterResp, err = s.bookChapterRepo.AddByBookID(ctx, bookChapter)
		if err != nil {
			return err
		}

		bookJSON, err := json.Marshal(bookChapter)
		if err != nil {
			return err
		}

		outbox := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      model.TableNameBookChapter,
			Operation: model.DatabaseOperationCreate,
			Payload:   bookJSON,
			Status:    model.OutboxStatusCreated,
			CreatedAt: time.Now(),
		}

		return s.outboxRepo.AddEvent(ctx, tx, outbox)
	})
	if err != nil {
		return nil, err
	}

	s.pool.SubmitCtx(constant.DoneServerCtx, agilePool.TaskFunc(func() error {
		err := addNewChapterFeedTask(
			constant.DoneServerCtx,
			&s,
			book,
			bookChapter,
			s.db,
		)
		if err != nil {
			logger.Error("add new chapter feed fail", zap.Error(err))
		}
		return err
	}))

	return toBookChapterResponse(s.cfg, bookChapterResp, book), nil

}

func addNewChapterFeedTask(
	ctx context.Context,
	s *bookChapterService,
	book *model.Book,
	chapter *model.BookChapter,
	db *gorm.DB,
) error {
	userIDs, err := s.userBookshelfRepo.ListUserIDsByBookID(ctx, s.db, book.ID)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}

	feeds := buildNewChapterFeeds(book, chapter, userIDs)
	if len(feeds) > 50 {
		return s.feedClient.PublishFeeds(ctx, s.cfg, feeds, userIDs)
	}

	if _, err := s.feedRepo.CreateBatchIgnoreConflict(ctx, db, feeds); err != nil {
		return err
	}
	if err := s.feedCursorRepo.BatchUpdateUnreadCount(ctx, db, userIDs, 1); err != nil {
		return err
	}
	return warmFeedCache(ctx, s.feedCache, feeds)
}

func buildNewChapterFeeds(
	book *model.Book,
	chapter *model.BookChapter,
	userIDs []string,
) []*model.FeedInbox {
	now := time.Now()
	chapterID := chapter.ID
	chapterTitle := chapter.Title
	feeds := make([]*model.FeedInbox, 0, len(userIDs))

	for _, userID := range userIDs {
		feeds = append(feeds, &model.FeedInbox{
			ID:           uuid.New().String(),
			UserID:       userID,
			EventType:    model.FeedEventNewChapter,
			BookID:       book.ID,
			ChapterID:    &chapterID,
			BookName:     book.Name,
			BookCover:    book.CoverImage,
			ChapterTitle: &chapterTitle,
			Content:      fmt.Sprintf("《%s》更新了章节：%s", book.Name, chapter.Title),
			IsRead:       false,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	return feeds
}

func warmFeedCache(ctx context.Context, feedCache cache.FeedCache, feeds []*model.FeedInbox) error {
	userFeeds := make(map[string][]*model.FeedInbox)
	userIDs := make([]string, 0)
	for _, feed := range feeds {
		if _, ok := userFeeds[feed.UserID]; !ok {
			userIDs = append(userIDs, feed.UserID)
		}
		userFeeds[feed.UserID] = append(userFeeds[feed.UserID], feed)
	}

	if err := feedCache.BatchIncrUnreadCount(ctx, userIDs, 1); err != nil {
		return err
	}
	for userID, feeds := range userFeeds {
		if err := feedCache.PushFeeds(ctx, userID, feeds); err != nil {
			return err
		}
	}
	return nil
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

func (s *bookChapterService) GetChapter(ctx context.Context, userID string, bookChapterID string) (*dto.BookChapterResponse, error) {
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
	s.pool.SubmitCtx(constant.DoneServerCtx, agilePool.TaskFunc(func() error {
		return bookThermalCacheAddByReadTask(constant.DoneServerCtx, s, userID, book, bookChapter)
	}))

	return toBookChapterResponse(s.cfg, bookChapter, book), nil
}

func bookThermalCacheAddByReadTask(ctx context.Context, s *bookChapterService, userID string, book *model.Book, bookChapter *model.BookChapter) error {
	var err error
	// 用户添加对应书籍数量
	_, _ = s.userStatCache.AddReadBook(ctx, userID, book.ID)

	// 检查锁：阅读频次需要大于1min
	exist, _ := s.limitCache.CheckUserBrowse1Min(ctx, userID)
	if exist {
		return nil
	}

	// 用户添加对应阅读量
	pageWords := int64(utils.CountPassageWord(bookChapter.Content))
	// todo perf:性能优化
	addReadWordsErr := s.userStatCache.IncreaseReadWordsNum(ctx, userID, pageWords)
	if addReadWordsErr != nil {
		logger.Fatal("increase read words fail: %v", zap.String("bookChapterId", bookChapter.ID))
	}

	// 检查锁：每阅读5min记录到书籍
	exist, _ = s.limitCache.CheckUserBrowseBook5Min(ctx, userID, book.ID)
	if exist {
		return nil
	}
	// 添加锁：热力值限流
	_ = s.limitCache.AddUserBrowseBook5Min(ctx, userID, book.ID)
	_ = s.limitCache.AddUserBrowse1Min(ctx, userID)

	// 数据库添加数据
	// todo perf:性能优化
	_, err = s.bookRepo.IncreaseOfBrowseAmount(ctx, s.db, book.ID)
	if err != nil {
		return err
	}

	// 更新数据
	s.pool.SubmitCtx(constant.DoneServerCtx, agilePool.TaskFunc(func() error {
		return bookThermalCacheAddValueTask(constant.DoneServerCtx, s.db, s.bookThermalCache, s.bookCache, s.bookRepo, s.pool, book.ID, 1)
	}))

	return nil
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

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.bookChapterRepo.Update(ctx, old.ID, cur)
		if err != nil {
			return err
		}

		bookJSON, err := json.Marshal(cur)
		if err != nil {
			return err
		}

		outbox := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      model.TableNameBookChapter,
			Operation: model.DatabaseOperationUpdate,
			Payload:   bookJSON,
			Status:    model.OutboxStatusCreated,
			CreatedAt: time.Now(),
		}

		return s.outboxRepo.AddEvent(ctx, tx, outbox)
	})
	if err != nil {
		return nil, err
	}

	// 更新缓存
	_ = s.bookChapterCache.UpdateBookChapter(ctx, cur)

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

	// 检查书籍所有权
	if book.OwnerID != userID {
		return ErrPermission
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 删除章节
		err = s.bookChapterRepo.Delete(ctx, chapterID)
		if err != nil {
			return err
		}

		// 构建删除事件的 payload
		deletePayload := map[string]string{"id": chapterID}
		bookJSON, err := json.Marshal(deletePayload)
		if err != nil {
			return err
		}

		outbox := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      model.TableNameBookChapter,
			Operation: model.DatabaseOperationDelete,
			Payload:   bookJSON,
			Status:    model.OutboxStatusCreated,
			CreatedAt: time.Now(),
		}

		return s.outboxRepo.AddEvent(ctx, tx, outbox)
	})
	if err != nil {
		return err
	}

	return nil
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

// SearchBookChaptersV2 v2 版本的搜索章节方法，使用 Elasticsearch
func (s bookChapterService) SearchBookChaptersV2(ctx context.Context, req *dto.SearchBookChaptersV2Request) (*dto.SearchBookChaptersV2Response, error) {
	filter := model.BookChapterFilterWithIDPage{
		Query:          req.Query,
		BookID:         req.BookID,
		CreatedAtStart: req.CreatedAtStart,
		CreatedAtEnd:   req.CreatedAtEnd,
		PriceStart:     req.PriceStart,
		PriceEnd:       req.PriceEnd,
		Order:          req.Order,
		OrderDirection: req.OrderDirection,
		IDPage: model.IDPage{
			LastID: req.LastID,
			Limit:  req.Limit,
		},
	}

	if req.VIPLevel != "" {
		filter.VIPLevel = strings.Split(req.VIPLevel, ",")
	}

	esResults, err := s.bookChapterESRepo.Search(ctx, filter)
	if err != nil {
		return nil, err
	}

	results := make([]*dto.BookChapterSearchResult, 0, len(esResults))
	for _, esResult := range esResults {
		var book *model.Book
		var exist bool
		var err error

		// 查询关联的书籍信息以组合返回
		book, exist, _ = s.bookCache.GetBook(ctx, esResult.BookChapter.BookID)
		if !exist {
			book, err = s.bookRepo.GetByID(ctx, esResult.BookChapter.BookID)
			if err != nil {
				logger.Error("SearchBookChaptersV2 GetByID err: ", zap.Error(err))
				continue
			}
			if book != nil {
				_ = s.bookCache.UpdateBook(ctx, book)
			}
		}

		if book != nil {
			chapterResp := toBookChapterResponse(s.cfg, &esResult.BookChapter, book)
			results = append(results, &dto.BookChapterSearchResult{
				BookChapter: *chapterResp,
				Highlight:   esResult.Highlight,
			})
		}
	}

	return &dto.SearchBookChaptersV2Response{
		Chapters: results,
	}, nil
}
