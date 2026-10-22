package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/response"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	feedRepo                 repository.FeedRepository // Deprecated: 旧版 feed 仓储，将被移除
	outboxRepo               repository.OutboxRepository
	bookCache                cache.BookCache
	bookThermalCache         cache.BookThermalCache
	bookChapterCache         cache.BookChapterCache
	userStatCache            cache.UserStatCache
	limitCache               cache.LimitCache
	feedCache                cache.FeedCache           // Deprecated: 旧版 feed 缓存，将被移除
	feedClient               kafka_client.FeedClientV1 // Deprecated: 旧版 feed Kafka 客户端，将被移除
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
	outboxRepo repository.OutboxRepository,
	bookCache cache.BookCache,
	bookThermalCache cache.BookThermalCache,
	bookChapterCache cache.BookChapterCache,
	userStatCache cache.UserStatCache,
	limitCache cache.LimitCache,
	feedCache cache.FeedCache,
	feedClient kafka_client.FeedClientV1,
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

func (s *bookChapterService) AddByBookID(ctx context.Context, userID string, req *dto.CreatBookChapterRequest) (*dto.BookChapterResponse, error) {
	book, err := s.bookRepo.GetByID(ctx, req.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, response.ErrBookNotFound
	}

	if book.OwnerID != userID {
		return nil, response.ErrPermission
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
			s,
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

func (s *bookChapterService) GuestGetChapter(ctx context.Context, chapterID string) (*dto.BookChapterResponse, error) {
	var err error
	bookChapter, err := queryBookChapterDetailWithCache(ctx, s.db, s.bookChapterCache, s.bookChapterRepo, chapterID)

	if err != nil {
		return nil, err
	}
	if bookChapter == nil {
		return nil, response.ErrBookChapterNotFound
	}

	book, err := queryBookDetailWithCache(ctx, s.db, s.bookCache, s.bookRepo, bookChapter.BookID)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, response.ErrBookNotFound
	}

	if VIPStrToLevel(bookChapter.VIPLevel) > VIPStrToLevel("") {
		return nil, response.ErrUserNotLogin
	}
	return toBookChapterResponse(s.cfg, bookChapter, book), nil
}

func (s *bookChapterService) GetChapter(ctx context.Context, userID string, bookChapterID string) (*dto.BookChapterResponse, error) {
	user, err := s.userRepo.GetByID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, response.ErrUserNotFound
	}

	bookChapterDetail, err := queryBookChapterDetailWithCache(ctx, s.db, s.bookChapterCache, s.bookChapterRepo, bookChapterID)
	if err != nil {
		return nil, err
	}
	if bookChapterDetail == nil {
		return nil, response.ErrBookChapterNotFound
	}

	bookDetail, err := queryBookDetailWithCache(ctx, s.db, s.bookCache, s.bookRepo, bookChapterDetail.BookID)
	if err != nil {
		return nil, err
	}
	if bookDetail == nil {
		return nil, response.ErrBookNotFound
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
		if VIPStrToLevel(bookChapterDetail.VIPLevel) > VIPStrToLevel(userAsset.VIPLevel) {
			return nil, response.ErrVIPLevel
		}
	}

	// 向redis里面添加热力值
	s.pool.SubmitCtx(constant.DoneServerCtx, agilePool.TaskFunc(func() error {
		return bookThermalCacheAddByReadTask(constant.DoneServerCtx, s, userID, bookDetail, bookChapterDetail)
	}))

	return toBookChapterResponse(s.cfg, bookChapterDetail, bookDetail), nil
}

func (s *bookChapterService) UpdateChapter(ctx context.Context, userID string, req *dto.UpdateBookChapterRequest) (*dto.BookChapterResponse, error) {
	var err error
	bookChapterDetail, err := queryBookChapterDetailWithCache(ctx, s.db, s.bookChapterCache, s.bookChapterRepo, req.ID)
	if err != nil {
		return nil, err
	}
	if bookChapterDetail == nil {
		return nil, response.ErrBookChapterNotFound
	}

	bookDetail, err := queryBookDetailWithCache(ctx, s.db, s.bookCache, s.bookRepo, bookChapterDetail.BookID)
	if err != nil {
		return nil, err
	}
	if bookDetail == nil {
		return nil, response.ErrBookNotFound
	}

	if bookDetail.OwnerID != userID {
		return nil, response.ErrPermission
	}

	var curBookChapterDetail = *bookChapterDetail

	// 更改BookChapterInfo
	var isAlterBookChapterInfo bool = false
	curBookChapterInfo := &model.BookChapterInfo{
		ID:       bookChapterDetail.ID,
		BookID:   bookChapterDetail.BookID,
		Title:    bookChapterDetail.Title,
		Order:    bookChapterDetail.Order,
		VIPLevel: bookChapterDetail.VIPLevel,
		Price:    bookChapterDetail.Price,
	}

	if req.Title != nil && *req.Title != "" {
		isAlterBookChapterInfo = true
		curBookChapterInfo.Title = *req.Title
		curBookChapterDetail.Title = *req.Title
	}
	if req.VIPLevel != nil {
		isAlterBookChapterInfo = true
		curBookChapterInfo.VIPLevel = VIPLevelToStr(VIPStrToLevel(*req.VIPLevel))
		curBookChapterDetail.VIPLevel = VIPLevelToStr(VIPStrToLevel(*req.VIPLevel))
	}
	if req.Price != nil {
		isAlterBookChapterInfo = true
		curBookChapterInfo.Price = *req.Price
		curBookChapterDetail.Price = *req.Price
	}

	if isAlterBookChapterInfo {
		// 合并完整章节数据作为 outbox 事件载荷
		mergeBookChapter := &model.BookChapter{
			ID:        curBookChapterInfo.ID,
			BookID:    curBookChapterInfo.BookID,
			Title:     curBookChapterInfo.Title,
			Content:   bookChapterDetail.Content,
			Order:     curBookChapterInfo.Order,
			VIPLevel:  curBookChapterInfo.VIPLevel,
			Price:     curBookChapterInfo.Price,
			CreatedAt: curBookChapterInfo.CreatedAt,
			UpdatedAt: curBookChapterInfo.UpdatedAt,
			DeletedAt: curBookChapterInfo.DeletedAt,
		}
		mergeBookChapterJSON, err := json.Marshal(mergeBookChapter)
		if err != nil {
			return nil, fmt.Errorf("merge book_chapter_info fail")
		}
		outboxDetail := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      mergeBookChapter.TableName(),
			Operation: model.DatabaseOperationUpdate,
			Payload:   mergeBookChapterJSON,
		}

		// 先更新数据库（事务内写入 outbox 事件）
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := s.bookChapterRepo.BookChapterInfoUpdate(ctx, tx, curBookChapterInfo.ID, curBookChapterInfo); err != nil {
				return err
			}
			return s.outboxRepo.AddEvent(ctx, tx, outboxDetail)
		})
		if err != nil {
			return nil, err
		}
		// 再删除缓存
		if err := s.bookChapterCache.DeleteBookChapterInfo(ctx, curBookChapterInfo.ID); err != nil {
			logger.Warn("delete book_chapter_info cache fail", zap.Error(err))
		}
	}

	// 更改BookChapterContent
	var isAlterBookChapterContent bool = false
	curBookChapterContent := &model.BookChapterContent{
		ID:      bookChapterDetail.ID,
		Content: bookChapterDetail.Content,
	}

	if req.Content != nil && *req.Content != "" {
		isAlterBookChapterContent = true
		curBookChapterContent.Content = *req.Content
		curBookChapterDetail.Content = *req.Content
	}

	if isAlterBookChapterContent {
		// 合并完整章节数据作为 outbox 事件载荷
		mergeBookChapter := &model.BookChapter{
			ID:        curBookChapterContent.ID,
			BookID:    bookChapterDetail.BookID,
			Title:     bookChapterDetail.Title,
			Content:   curBookChapterContent.Content,
			Order:     bookChapterDetail.Order,
			VIPLevel:  bookChapterDetail.VIPLevel,
			Price:     bookChapterDetail.Price,
			CreatedAt: bookChapterDetail.CreatedAt,
			UpdatedAt: bookChapterDetail.UpdatedAt,
			DeletedAt: bookChapterDetail.DeletedAt,
		}
		mergeBookChapterJSON, err := json.Marshal(mergeBookChapter)
		if err != nil {
			return nil, fmt.Errorf("merge book_chapter_content fail")
		}
		outboxDetail := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      mergeBookChapter.TableName(),
			Operation: model.DatabaseOperationUpdate,
			Payload:   mergeBookChapterJSON,
		}

		// 先更新数据库（事务内写入 outbox 事件）
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := s.bookChapterRepo.BookChapterContentUpdate(ctx, tx, curBookChapterContent.ID, curBookChapterContent); err != nil {
				return err
			}
			return s.outboxRepo.AddEvent(ctx, tx, outboxDetail)
		})
		if err != nil {
			return nil, err
		}
		// 再删除缓存
		if err := s.bookChapterCache.DeleteBookChapterContent(ctx, curBookChapterContent.ID); err != nil {
			logger.Warn("delete book_chapter_content cache fail", zap.Error(err))
		}
	}

	return toBookChapterResponse(s.cfg, &curBookChapterDetail, bookDetail), nil
}

func (s *bookChapterService) DeleteChapter(ctx context.Context, userID string, chapterID string) error {
	// 检查章节是否存在
	chapter, err := s.bookChapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return response.ErrBookChapterNotFound
	}

	// 检查书籍是否存在
	book, err := s.bookRepo.GetByID(ctx, chapter.BookID)
	if err != nil {
		return err
	}
	if book == nil {
		return response.ErrBookNotFound
	}

	// 检查书籍所有权
	if book.OwnerID != userID {
		return response.ErrPermission
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

func (s *bookChapterService) GetChapterList(ctx context.Context, req *dto.GetBookChapterListRequest) (*dto.GetBookChapterListResponse, error) {
	exist, err := s.bookRepo.GetByID(ctx, req.BookID)
	if err != nil {
		return nil, err
	}
	if exist == nil {
		return nil, response.ErrBookNotFound
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

func (s *bookChapterService) BuyBookChapter(ctx context.Context, userID string, req *dto.BuyBookChapterRequest) error {
	bookChapter, err := s.bookChapterRepo.GetByID(ctx, req.ChapterID)
	if err != nil {
		return err
	}
	if bookChapter == nil {
		return response.ErrBookNotFound
	}

	if bookChapter.Price != req.Price {
		return response.ErrParam
	}
	if bookChapter.BookID != req.BookID {
		return response.ErrParam
	}

	tx := s.db.Begin()
	exist, err := s.userPurchasedChapterRepo.GetByChapterIDAndUserID(ctx, req.ChapterID, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if exist != nil {
		tx.Rollback()
		return response.ErrBookAlreadyBuy
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
		return response.ErrUserNotFound
	}
	if userAsset.Gold < req.Price {
		tx.Rollback()
		return response.ErrSufficientFunds
	}

	_, err = s.userAssetRepo.SubGoldByID(ctx, tx, userID, req.Price)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 结算创作者收益：所付金币折半（向上取整）计入作者资产
	creatorGold := calcCreatorRevenue(req.Price)
	if creatorGold > 0 {
		book, err := s.bookRepo.GetByID(ctx, req.BookID)
		if err != nil {
			tx.Rollback()
			return err
		}
		if book == nil {
			tx.Rollback()
			return response.ErrBookNotFound
		}
		_, err = s.userAssetRepo.AddGoldByID(ctx, tx, book.OwnerID, creatorGold)
		if err != nil {
			tx.Rollback()
			return err
		}
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

func (s *bookChapterService) GetChapterInfo(ctx context.Context, bookChapterID string) (*dto.BookChapterInfoResponse, error) {
	bookChapterInfo, err := queryBookChapterInfoWithCache(ctx, s.db, s.bookChapterCache, s.bookChapterRepo, bookChapterID)
	if err != nil {
		return nil, err
	}
	if bookChapterInfo == nil {
		return nil, response.ErrBookChapterNotFound
	}

	bookDetail, err := queryBookDetailWithCache(ctx, s.db, s.bookCache, s.bookRepo, bookChapterInfo.BookID)
	if err != nil {
		return nil, err
	}
	if bookDetail == nil {
		return nil, response.ErrBookNotFound
	}

	return &dto.BookChapterInfoResponse{
		ChapterID: bookChapterInfo.ID,
		Title:     bookChapterInfo.Title,
		Order:     bookChapterInfo.Order,
		VIPLevel:  bookChapterInfo.VIPLevel,
		Price:     bookChapterInfo.Price,
		Book:      toBookResponse(s.cfg, bookDetail),
	}, nil
}

// SearchBookChaptersV2 v2 版本的搜索章节方法，使用 Elasticsearch
func (s *bookChapterService) SearchBookChaptersV2(ctx context.Context, req *dto.SearchBookChaptersV2Request) (*dto.SearchBookChaptersV2Response, error) {
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
		// 查询关联的书籍信息以组合返回
		bookDetail, err := queryBookDetailWithCache(ctx, s.db, s.bookCache, s.bookRepo, esResult.BookChapter.BookID)
		if err != nil {
			logger.Error("book_detail query failed", zap.Error(err))
			continue
		}
		if bookDetail == nil {
			logger.Warn("book_detail is nil", zap.Error(err))
			continue
		}

		chapterResp := toBookChapterResponse(s.cfg, &esResult.BookChapter, bookDetail)
		results = append(results, &dto.BookChapterSearchResult{
			BookChapter: *chapterResp,
			Highlight:   esResult.Highlight,
		})
	}

	return &dto.SearchBookChaptersV2Response{
		Chapters: results,
	}, nil
}
