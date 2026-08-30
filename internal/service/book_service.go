package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/parameter"
	"fiction-book/pkg/response"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BookService 书籍服务接口
type BookService interface {
	Create(ctx context.Context, userID string, req *dto.CreateBookRequest) (*dto.BookResponse, error)
	GetByID(ctx context.Context, id string) (*dto.BookResponse, error)
	List(ctx context.Context, page int, size int) (*dto.BookListResponse, error)
	ListBooks(ctx context.Context, req *dto.ListBooksRequest) (*dto.BookListResponse, error)
	BuyBook(ctx context.Context, userID string, bookID string, req *dto.BuyBookRequest) (*dto.GetUserPurchasedBookResponse, error)
	UploadBookCover(ctx context.Context, userID string, req *dto.UploadBookCoverRequest) (*dto.UploadBookCoverResponse, error)
	GetBookListByThermalDaily(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error)
	GetListByOwnerID(ctx context.Context, userID string, req *dto.GetBookListByOwnerIDRequest) (*dto.BookListResponse, error)
	GetReadHistoryRecently(ctx context.Context, userID string, page int, size int) (*dto.BookListResponse, error)
	UpdateBook(ctx context.Context, userID string, req *dto.UpdateBookRequest) (*dto.BookResponse, error)
	UpdateBookTag(ctx context.Context, userID string, req *dto.UpdateBookTagRequest) (*dto.UpdateBookTagResponse, error)
	GetBookTags(ctx context.Context, bookID string) (*dto.GetBookTagResponse, error)
	GetBookListByThermalWeekly(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error)
	GetBookListByThermalMonthly(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error)
	SearchBooksV2(ctx context.Context, req *dto.SearchBooksV2Request) (*dto.SearchBooksV2Response, error)
}

type bookService struct {
	bookRepo          repository.BookRepository
	bookTagRepo       repository.BookTagRepository
	bookESRepo        repository.BookESRepository
	userRepo          repository.UserRepository
	userAssetRepo     repository.UserAssetRepository
	purchasedBookRepo repository.UserPurchasedBookRepository
	outboxRepo        repository.OutboxRepository
	userInfoRepo      repository.UserInfoRepository
	userFeedRepo      repository.UserFeedRepository
	bookCache         cache.BookCache
	userStatCache     cache.UserStatCache
	bookThermalCache  cache.BookThermalCache
	bookTagCache      cache.BookTagCache
	userInfoCache     cache.UserInfoCache
	userFeedCache     cache.UserFeedCache
	feedClient        kafka_client.FeedClient
	db                *gorm.DB
	cfg               *config.Config
	pool              *agilePool.Pool
}

// NewBookService 创建书籍服务实例
func NewBookService(
	bookRepo repository.BookRepository,
	bookTagRepo repository.BookTagRepository,
	bookESRepo repository.BookESRepository,
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	purchasedBookRepo repository.UserPurchasedBookRepository,
	outboxRepo repository.OutboxRepository,
	userInfoRepo repository.UserInfoRepository,
	userFeedRepo repository.UserFeedRepository,
	bookCache cache.BookCache,
	userStatCache cache.UserStatCache,
	bookThermalCache cache.BookThermalCache,
	bookTagCache cache.BookTagCache,
	userInfoCache cache.UserInfoCache,
	userFeedCache cache.UserFeedCache,
	feedClient kafka_client.FeedClient,
	db *gorm.DB,
	cfg *config.Config,
	pool *agilePool.Pool,
) BookService {
	return &bookService{
		bookRepo:          bookRepo,
		bookTagRepo:       bookTagRepo,
		bookESRepo:        bookESRepo,
		userRepo:          userRepo,
		userAssetRepo:     userAssetRepo,
		purchasedBookRepo: purchasedBookRepo,
		outboxRepo:        outboxRepo,
		userInfoRepo:      userInfoRepo,
		userFeedRepo:      userFeedRepo,
		bookCache:         bookCache,
		userStatCache:     userStatCache,
		bookThermalCache:  bookThermalCache,
		bookTagCache:      bookTagCache,
		userInfoCache:     userInfoCache,
		userFeedCache:     userFeedCache,
		feedClient:        feedClient,
		db:                db,
		cfg:               cfg,
		pool:              pool,
	}
}

func (s *bookService) Create(ctx context.Context, userID string, req *dto.CreateBookRequest) (*dto.BookResponse, error) {
	existingBook, err := s.bookRepo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if existingBook != nil {
		return nil, response.ErrBookExists
	}

	userInfoDetail, err := queryUserInfoDetailWithCache(ctx, s.db, s.userInfoCache, s.userInfoRepo, userID)
	if err != nil {
		return nil, err
	}
	if userInfoDetail == nil {
		return nil, response.ErrUserNotFound
	}

	book := &model.Book{
		ID:              uuid.New().String(),
		OwnerID:         userID,
		Name:            req.Name,
		Description:     req.Description,
		VIPLevel:        req.VIPLevel,
		Price:           req.Price,
		CoverImage:      req.CoverImage,
		PushedAt:        req.PushAt,
		BrowseAmount:    0,
		BookshelfAmount: 0,
	}

	calculateRank := func(userInfoDetail *model.UserInfo) int64 {
		return int64(userInfoDetail.IsV*10) + userInfoDetail.FansNum/100
	}

	userFeedDetail := &model.UserFeed{
		FeedID:    uuid.New().String(),
		UserID:    userID,
		BookID:    book.ID,
		Content:   fmt.Sprintf("发布了书籍%s", req.Name),
		Rank:      calculateRank(userInfoDetail),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		// 写入数据
		err = s.bookRepo.Create(ctx, tx, book)
		if err != nil {
			return err
		}

		// 更新用户数据
		err = s.userInfoRepo.IncrementReadBookNum(ctx, tx, userID, 1)
		if err != nil {
			return err
		}

		// 添加Feed消息
		err = s.userFeedRepo.CreateUserFeed(ctx, tx, userFeedDetail)
		if err != nil {
			return err
		}

		// 同步到ES
		bookJSON, err := json.Marshal(book)
		if err != nil {
			return err
		}

		outbox := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      (&model.Book{}).TableName(),
			Operation: model.DatabaseOperationCreate,
			Payload:   bookJSON,
			Status:    model.OutboxStatusCreated,
			CreatedAt: time.Now(),
		}

		err = s.outboxRepo.AddEvent(ctx, tx, outbox)
		if err != nil {
			return err
		}
		return nil
	})

	if userInfoDetail.IsV == 0 {
		// 小V用户:feed流写推送
		err := s.feedClient.PublishFeed(ctx, *userFeedDetail)
		if err != nil {
			logger.Error("error publishing feed", zap.Error(err))
			return nil, nil
		}
	}

	if err != nil {
		return nil, err
	}
	return toBookResponse(s.cfg, book), nil
}

func toBookResponse(cfg *config.Config, book *model.Book) *dto.BookResponse {
	return &dto.BookResponse{
		ID:              book.ID,
		Name:            book.Name,
		Description:     book.Description,
		CoverImage:      getImageUrl(cfg, book.CoverImage, "/upload/book_cover/"),
		OwnerID:         book.OwnerID,
		VIPLevel:        book.VIPLevel,
		Price:           book.Price,
		PushAt:          book.PushedAt.Unix(),
		BrowseAmount:    book.BrowseAmount,
		BookshelfAmount: book.BookshelfAmount,
		CreatedAt:       book.CreatedAt.Unix(),
		UpdatedAt:       book.UpdatedAt.Unix(),
	}
}

func getImageUrl(cfg *config.Config, path string, prefix string) string {
	if path == "" {
		return fmt.Sprintf("%s%s%s", cfg.Server.Website, prefix, "default.jpg")
	}
	return fmt.Sprintf("%s%s", cfg.Server.Website, path)
}

func (s *bookService) GetByID(ctx context.Context, id string) (*dto.BookResponse, error) {
	book, err := s.bookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if book == nil {
		return nil, response.ErrBookNotFound
	}
	return toBookResponse(s.cfg, book), nil
}

// List 列出所有的书籍
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
		bookResponses = append(bookResponses, toBookResponse(s.cfg, book))
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

// ListBooks 根据排行列举书籍
func (s *bookService) ListBooks(ctx context.Context, req *dto.ListBooksRequest) (*dto.BookListResponse, error) {
	books, err := s.bookRepo.ListWithOption(ctx, repository.BookListOption{
		Page:          req.Page,
		Size:          req.Size,
		SortKey:       req.SortKey,
		SortOrder:     req.SortOrder,
		PushStartTime: req.PushStartTime,
		PushEndTime:   req.PushEndTime,
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
		bookResponses = append(bookResponses, toBookResponse(s.cfg, book))
	}
	if len(bookResponses) == 0 {
		bookResponses = make([]*dto.BookResponse, 0)
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

func (s *bookService) BuyBook(ctx context.Context, userID string, bookID string, req *dto.BuyBookRequest) (*dto.GetUserPurchasedBookResponse, error) {
	err := s.db.Transaction(func(db *gorm.DB) error {
		// 获取书籍信息
		book, err := s.bookRepo.GetByID(ctx, bookID)
		if err != nil {
			return err
		}
		if book == nil {
			return response.ErrBookNotFound
		}

		// 获取用户资产
		userAsset, err := s.userAssetRepo.GetByID(ctx, db, userID)
		if err != nil {
			return err
		}
		if userAsset == nil {
			return response.ErrUserNotFound
		}

		if req.Price != book.Price {
			return response.ErrParam
		}

		// 判断是否已经购买
		exist, _, err := s.purchasedBookRepo.CheckPurchasedBook(ctx, userID, bookID)
		if err != nil {
			return err
		}
		if exist {
			return response.ErrBookAlreadyBuy
		}

		// 判断资产是否足够
		if userAsset.Gold < book.Price {
			return response.ErrSufficientFunds
		}

		// 获取资产
		userAssetNow, err := s.userAssetRepo.SubGoldByID(ctx, db, userID, book.Price)
		if err != nil {
			return err
		}
		if userAssetNow == nil {
			return response.ErrUserNotFound
		}
		if userAsset.Gold-userAssetNow.Gold != book.Price {
			return errors.New("资产扣费失败")
		}

		// 添加已购买书籍
		purchasedBook := &model.UserPurchasedBook{
			ID:       uuid.New().String(),
			UserID:   userID,
			BookID:   bookID,
			ActiveAt: time.Now().Add(time.Hour * 24 * 30 * 12 * 3),
		}
		err = s.purchasedBookRepo.AddPurchasedBook(ctx, purchasedBook)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 获取购买信息
	_, purchased, err := s.purchasedBookRepo.CheckPurchasedBook(ctx, userID, bookID)
	if err != nil {
		return nil, err
	}

	return toUserPurchasedBookResponse(purchased), nil
}

func (s *bookService) UploadBookCover(ctx context.Context, userID string, req *dto.UploadBookCoverRequest) (*dto.UploadBookCoverResponse, error) {
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

	imgId := getFileID(book.CoverImage, "/upload/book_cover/")
	if imgId == "" {
		return nil, errors.New("book_cover_id decode error")
	}

	uploadOut, err := uploadImage(&parameter.UploadImageIn{
		Avatar:     req.CoverImage,
		ImageId:    imgId,
		ImagePath:  "/upload/book_cover/",
		SizeWidth:  368,
		SizeHeight: 460,
		SizeFile:   10 * 1024 * 1024,
		AcceptType: map[string]bool{
			"jpg": true,
		},
	})
	if err != nil {
		return nil, err
	}

	book.CoverImage = uploadOut.FilePath
	err = s.db.Transaction(func(tx *gorm.DB) error {
		err := s.bookRepo.Update(ctx, tx, book)
		if err != nil {
			return err
		}

		bookJSON, err := json.Marshal(book)
		if err != nil {
			return err
		}

		outbox := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      (&model.Book{}).TableName(),
			Operation: model.DatabaseOperationUpdate,
			Payload:   bookJSON,
			Status:    model.OutboxStatusCreated,
			CreatedAt: time.Now(),
		}

		err = s.outboxRepo.AddEvent(ctx, tx, outbox)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.UploadBookCoverResponse{
		BookID: book.ID,
		Url:    fmt.Sprintf("%s%s", s.cfg.Server.Website, uploadOut.FilePath),
	}, nil
}

func (s *bookService) GetBookListByThermalDaily(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error) {
	bookListResponse := make([]*dto.BookResponse, 0)
	pageResponse := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: -1,
	}
	res := &dto.BookListResponse{
		Books:            bookListResponse,
		PageListResponse: pageResponse,
	}

	// 当前页数的排行榜缓存失效时
	rankRes, err := s.bookCache.GetBookThermalDaily(ctx, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	// 逐行获取
	for _, bookRecord := range rankRes {
		// 尝试通过缓存获取
		book, bookCatchHit, _ := s.bookCache.GetBook(ctx, bookRecord)
		if bookCatchHit == false {
			// 当缓存未命中
			book, err = s.bookRepo.GetByID(ctx, bookRecord)
			if err != nil {
				return nil, err
			}
			if book == nil {
				logger.Error(fmt.Sprintf("error find book: %s", bookRecord))
				continue
			}
			_ = s.bookCache.UpdateBook(ctx, book.ID, book)
		}
		bookListResponse = append(bookListResponse, toBookResponse(s.cfg, book))
	}
	res.Books = bookListResponse

	return res, nil
}

func (s *bookService) GetListByOwnerID(ctx context.Context, userID string, req *dto.GetBookListByOwnerIDRequest) (*dto.BookListResponse, error) {
	bookListResponse := make([]*dto.BookResponse, 0)
	pageResponse := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: -1,
	}

	bookListModel, err := s.bookRepo.GetByOwnerID(ctx, userID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	for _, bookRecord := range bookListModel {
		bookListResponse = append(bookListResponse, toBookResponse(s.cfg, bookRecord))
	}

	bookListCount, err := s.bookRepo.CountByOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	pageResponse.Count = bookListCount

	res := &dto.BookListResponse{
		Books:            bookListResponse,
		PageListResponse: pageResponse,
	}
	return res, nil
}

func (s *bookService) GetReadHistoryRecently(ctx context.Context, userID string, page int, size int) (*dto.BookListResponse, error) {
	bookListResponse := make([]*dto.BookResponse, 0)
	pageResponse := dto.PageListResponse{
		Page:  page,
		Size:  size,
		Count: -1,
	}

	bookListCache, err := s.userStatCache.GetReadBookList(ctx, userID)
	if err != nil {
		return nil, err
	}

	startOffset := (page - 1) * size
	endOffset := startOffset + size
	if endOffset > len(bookListCache) {
		endOffset = len(bookListCache)
	}
	if startOffset > len(bookListCache) {
		startOffset = len(bookListCache)
	}

	bookListCache = bookListCache[startOffset:endOffset]

	itemsCount, _ := s.userStatCache.CountReadBookList(ctx, userID)
	pageResponse.Count = int(itemsCount)

	for _, bookRecord := range bookListCache {
		bookCache, bookCacheHit, _ := s.bookCache.GetBook(ctx, bookRecord)
		if bookCacheHit == false {
			bookCache, err = s.bookRepo.GetByID(ctx, bookRecord)
			if err != nil {
				return nil, err
			}
			_ = s.bookCache.UpdateBook(ctx, bookCache.ID, bookCache)
		}
		bookListResponse = append(bookListResponse, toBookResponse(s.cfg, bookCache))
	}

	historyResponse := &dto.BookListResponse{
		Books:            bookListResponse,
		PageListResponse: pageResponse,
	}
	return historyResponse, nil
}

func (s *bookService) UpdateBook(ctx context.Context, userID string, req *dto.UpdateBookRequest) (*dto.BookResponse, error) {
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

	if req.Name != nil && *req.Name != book.Name {
		book.Name = *req.Name

		// 检查是否重名
		exist, err := s.bookRepo.GetByName(ctx, book.Name)
		if err != nil {
			return nil, err
		}
		if exist != nil {
			return nil, response.ErrBookExists
		}
	}
	if req.Description != nil {
		book.Description = *req.Description
	}
	if req.Price != nil {
		book.Price = *req.Price
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		err := s.bookRepo.Update(ctx, tx, book)
		if err != nil {
			return err
		}

		bookJSON, err := json.Marshal(book)
		if err != nil {
			return err
		}

		outbox := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      (&model.Book{}).TableName(),
			Operation: model.DatabaseOperationUpdate,
			Payload:   bookJSON,
			Status:    model.OutboxStatusCreated,
			CreatedAt: time.Now(),
		}

		err = s.outboxRepo.AddEvent(ctx, tx, outbox)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	_ = s.bookCache.UpdateBook(ctx, book.ID, book)

	return toBookResponse(s.cfg, book), nil
}

func (s *bookService) GetBookListByThermalWeekly(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error) {
	bookListResponse := make([]*dto.BookResponse, 0)
	pageResponse := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: -1,
	}
	res := &dto.BookListResponse{
		Books:            bookListResponse,
		PageListResponse: pageResponse,
	}

	// 当前页数的排行榜缓存失效时
	rankRes, err := s.bookCache.GetBookThermalWeekly(ctx, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	bookListTemp, err := s.bookListToResponse(ctx, rankRes)
	if err != nil {
		return nil, err
	}

	if len(bookListTemp) == 0 {
		bookListTemp = make([]*dto.BookResponse, 0)
	}
	res.Books = bookListTemp

	return res, nil
}

func (s *bookService) bookListToResponse(ctx context.Context, rankRes []string) ([]*dto.BookResponse, error) {
	var bookResponse []*dto.BookResponse
	// 逐行获取
	for _, bookRecord := range rankRes {
		// 尝试通过缓存获取
		book, bookCatchHit, _ := s.bookCache.GetBook(ctx, bookRecord)
		if bookCatchHit == false {
			// 当缓存未命中
			var err error
			book, err = s.bookRepo.GetByID(ctx, bookRecord)
			if err != nil {
				return nil, err
			}
			if book == nil {
				logger.Error(fmt.Sprintf("error find book: %s", bookRecord))
				continue
			}
			_ = s.bookCache.UpdateBook(ctx, book.ID, book)
		}
		bookResponse = append(bookResponse, toBookResponse(s.cfg, book))
	}
	return bookResponse, nil
}

func (s *bookService) GetBookListByThermalMonthly(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error) {
	bookListResponse := make([]*dto.BookResponse, 0)
	pageResponse := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: -1,
	}
	res := &dto.BookListResponse{
		Books:            bookListResponse,
		PageListResponse: pageResponse,
	}

	// 当前页数的排行榜缓存失效时
	rankRes, err := s.bookCache.GetBookThermalMonthly(ctx, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	bookListTemp, err := s.bookListToResponse(ctx, rankRes)
	if err != nil {
		return nil, err
	}

	if len(bookListTemp) == 0 {
		bookListTemp = make([]*dto.BookResponse, 0)
	}
	res.Books = bookListTemp

	return res, nil
}

// UpdateBookTag 更新书籍标签（先删除旧标签，再批量插入新标签，并刷新缓存）
func (s *bookService) UpdateBookTag(ctx context.Context, userID string, req *dto.UpdateBookTagRequest) (*dto.UpdateBookTagResponse, error) {
	var book *model.Book
	var bookHit bool
	var err error

	// 验证书籍存在
	book, bookHit, _ = s.bookCache.GetBook(ctx, req.BookID)
	if !bookHit {
		var err error
		book, err = s.bookRepo.GetByID(ctx, req.BookID)
		if err != nil {
			return nil, err
		}
		if book == nil {
			return nil, response.ErrBookNotFound
		}
	}

	// 验证所有权
	if book.OwnerID != userID {
		return nil, response.ErrPermission
	}

	// 验证标签是否重复
	tagsSet := make(map[string]struct{})
	for _, tag := range req.Tags {
		if _, ok := tagsSet[tag]; ok {
			return nil, response.ErrParam
		}
		tagsSet[tag] = struct{}{}
	}

	// 校验标签合法性并构造 model 列表
	now := time.Now()
	tags := make([]model.BookTag, 0, len(req.Tags))
	for _, tagStr := range req.Tags {
		singleTag, err := model.StringToSingleBookTag(tagStr)
		if err != nil {
			return nil, response.ErrParam
		}
		tags = append(tags, model.BookTag{
			ID:        uuid.New().String(),
			BookID:    req.BookID,
			Tag:       singleTag,
			CreatedAt: now,
		})
	}

	// 事务：先删除旧标签，再批量插入新标签
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.bookTagRepo.DeleteAllByBookID(ctx, tx, req.BookID); err != nil {
			return err
		}
		return s.bookTagRepo.CreateBatch(ctx, tx, tags)
	})
	if err != nil {
		return nil, err
	}

	// 刷新 Redis 缓存（写入最新标签，TTL 1h）
	_ = s.bookTagCache.SetBookTags(ctx, req.BookID, tags)

	tagStrs := make([]string, 0, len(tags))
	for _, t := range tags {
		tagStrs = append(tagStrs, t.Tag.Tag())
	}
	return &dto.UpdateBookTagResponse{
		BookID: req.BookID,
		Tags:   tagStrs,
	}, nil
}

// GetBookTags 获取书籍标签（缓存优先，miss 时查 DB 并回填缓存）
func (s *bookService) GetBookTags(ctx context.Context, bookID string) (*dto.GetBookTagResponse, error) {
	// 验证书籍存在
	book, bookHit, _ := s.bookCache.GetBook(ctx, bookID)
	if !bookHit {
		var err error
		book, err = s.bookRepo.GetByID(ctx, bookID)
		if err != nil {
			return nil, err
		}
		if book == nil {
			return nil, response.ErrBookNotFound
		}
	}

	// 尝试从缓存获取
	cachedTags, hit, err := s.bookTagCache.GetBookTags(ctx, bookID)
	if err == nil && hit {
		tagStrs := make([]string, 0, len(cachedTags))
		for _, t := range cachedTags {
			tagStrs = append(tagStrs, t.Tag.Tag())
		}
		return &dto.GetBookTagResponse{BookID: bookID, Tags: tagStrs}, nil
	}

	// 缓存未命中，查数据库
	tags, err := s.bookTagRepo.ListByBookID(ctx, bookID)
	if err != nil {
		logger.Error("book tags get error", zap.String("book_id", bookID), zap.Error(err))
		return nil, err
	}

	// 回填缓存（忽略缓存写入错误）
	_ = s.bookTagCache.SetBookTags(ctx, bookID, tags)

	tagStrs := make([]string, 0, len(tags))
	for _, t := range tags {
		tagStrs = append(tagStrs, t.Tag.Tag())
	}
	return &dto.GetBookTagResponse{BookID: bookID, Tags: tagStrs}, nil
}

// SearchBooksV2 v2 版本的搜索方法，使用 Elasticsearch
func (s *bookService) SearchBooksV2(ctx context.Context, req *dto.SearchBooksV2Request) (*dto.SearchBooksV2Response, error) {
	// 转换为 model 的 BookFilterWithIDPage
	filter := &model.BookFilterWithIDPage{
		Query:                req.Query,
		PushedAtStart:        req.PushStartTime,
		PushedAtEnd:          req.PushEndTime,
		UpdatedAtStart:       req.UpdateStartTime,
		UpdatedAtEnd:         req.UpdateEndTime,
		PriceStart:           req.PriceStart,
		PriceEnd:             req.PriceEnd,
		OwnerID:              req.OwnerID,
		BookshelfAmountStart: req.BookshelfAmountStart,
		BookshelfAmountEnd:   req.BookshelfAmountEnd,
		ThermalValueStart:    req.ThermalValueStart,
		ThermalValueEnd:      req.ThermalValueEnd,
		Order:                req.Order,
		OrderDirection:       req.OrderDirection,
		IDPage: model.IDPage{
			LastID: req.LastID,
			Limit:  req.Limit,
		},
	}

	if req.VIPLevel != "" {
		filter.VIPLevel = strings.Split(req.VIPLevel, ",")
	}

	// 调用 ES 搜索
	esResults, err := s.bookESRepo.Search(ctx, filter)
	if err != nil {
		return nil, err
	}

	// 转换为响应
	results := make([]*dto.BookSearchResult, 0, len(esResults))
	for _, esResult := range esResults {
		bookResp := toBookResponse(s.cfg, &esResult.Book)
		results = append(results, &dto.BookSearchResult{
			Book:      *bookResp,
			Highlight: esResult.Highlight,
		})
	}

	return &dto.SearchBooksV2Response{
		Books: results,
	}, nil
}
