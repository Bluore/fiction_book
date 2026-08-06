package service

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/parameter"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrBookExists      = errors.New("book already exists")
	ErrBookNotFound    = errors.New("book not found")
	ErrParam           = errors.New("invalid param")
	ErrSufficientFunds = errors.New("sufficient funds")
	ErrBookAlreadyBuy  = errors.New("book already buy")
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
	GetBookListByThermalWeekly(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error)
	GetBookListByThermalMonthly(ctx context.Context, req *dto.GetBookListByThermalDaily) (*dto.BookListResponse, error)
	SearchBooksV2(ctx context.Context, req *dto.SearchBooksV2Request) (*dto.SearchBooksV2Response, error)
}

type bookService struct {
	bookRepo          repository.BookRepository
	bookESRepo        repository.BookESRepository
	userRepo          repository.UserRepository
	userAssetRepo     repository.UserAssetRepository
	purchasedBookRepo repository.UserPurchasedBookRepository
	outboxRepo        repository.OutboxRepository
	bookCache         cache.BookCache
	userStatCache     cache.UserStatCache
	bookThermalCache  cache.BookThermalCache
	db                *gorm.DB
	cfg               *config.Config
}

// NewBookService 创建书籍服务实例
func NewBookService(
	bookRepo repository.BookRepository,
	bookESRepo repository.BookESRepository,
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	purchasedBookRepo repository.UserPurchasedBookRepository,
	outboxRepo repository.OutboxRepository,
	bookCache cache.BookCache,
	userStatCache cache.UserStatCache,
	bookThermalCache cache.BookThermalCache,
	db *gorm.DB,
	cfg *config.Config,
) BookService {
	return &bookService{
		bookRepo:          bookRepo,
		bookESRepo:        bookESRepo,
		userRepo:          userRepo,
		userAssetRepo:     userAssetRepo,
		purchasedBookRepo: purchasedBookRepo,
		outboxRepo:        outboxRepo,
		bookCache:         bookCache,
		userStatCache:     userStatCache,
		bookThermalCache:  bookThermalCache,
		db:                db,
		cfg:               cfg,
	}
}

func (s *bookService) Create(ctx context.Context, userID string, req *dto.CreateBookRequest) (*dto.BookResponse, error) {
	existingBook, err := s.bookRepo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if existingBook != nil {
		return nil, ErrBookExists
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
	err = s.db.Transaction(func(tx *gorm.DB) error {
		err := s.bookRepo.Create(ctx, tx, book)
		if err != nil {
			return err
		}

		bookJSON, err := json.Marshal(book)
		if err != nil {
			return err
		}

		outbox := &model.Outbox{
			ID:        uuid.New().String(),
			Type:      model.Book{}.TableName(),
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
		return nil, ErrBookNotFound
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
			return ErrBookNotFound
		}

		// 获取用户资产
		userAsset, err := s.userAssetRepo.GetByID(ctx, db, userID)
		if err != nil {
			return err
		}
		if userAsset == nil {
			return ErrUserNotFound
		}

		if req.Price != book.Price {
			return ErrParam
		}

		// 判断是否已经购买
		exist, _, err := s.purchasedBookRepo.CheckPurchasedBook(ctx, userID, bookID)
		if err != nil {
			return err
		}
		if exist {
			return ErrBookAlreadyBuy
		}

		// 判断资产是否足够
		if userAsset.Gold < book.Price {
			return ErrSufficientFunds
		}

		// 获取资产
		userAssetNow, err := s.userAssetRepo.SubGoldByID(ctx, db, userID, book.Price)
		if err != nil {
			return err
		}
		if userAssetNow == nil {
			return ErrUserNotFound
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
		return nil, ErrBookNotFound
	}

	if book.OwnerID != userID {
		return nil, ErrPermission
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
			Type:      model.Book{}.TableName(),
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
			_ = s.bookCache.UpdateBook(ctx, book)
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
			_ = s.bookCache.UpdateBook(ctx, bookCache)
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
		return nil, ErrBookNotFound
	}

	if book.OwnerID != userID {
		return nil, ErrPermission
	}

	if req.Name != nil && *req.Name != book.Name {
		book.Name = *req.Name

		// 检查是否重名
		exist, err := s.bookRepo.GetByName(ctx, book.Name)
		if err != nil {
			return nil, err
		}
		if exist != nil {
			return nil, ErrBookExists
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
			Type:      model.Book{}.TableName(),
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

	_ = s.bookCache.UpdateBook(ctx, book)

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
			_ = s.bookCache.UpdateBook(ctx, book)
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
