package service

import (
	"context"
	"errors"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/parameter"
	"fmt"
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
}

type bookService struct {
	bookRepo          repository.BookRepository
	userRepo          repository.UserRepository
	userAssetRepo     repository.UserAssetRepository
	purchasedBookRepo repository.UserPurchasedBookRepository
	db                *gorm.DB
	cfg               *config.Config
}

// NewBookService 创建书籍服务实例
func NewBookService(
	bookRepo repository.BookRepository,
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	purchasedBookRepo repository.UserPurchasedBookRepository,
	db *gorm.DB,
	cfg *config.Config,
) BookService {
	return &bookService{
		bookRepo:          bookRepo,
		userRepo:          userRepo,
		userAssetRepo:     userAssetRepo,
		purchasedBookRepo: purchasedBookRepo,
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
	err = s.bookRepo.Create(ctx, book)
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
		PushAt:          book.PushedAt,
		BrowseAmount:    book.BrowseAmount,
		BookshelfAmount: book.BookshelfAmount,
		CreatedAt:       book.CreatedAt,
		UpdatedAt:       book.UpdatedAt,
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

// ListBooks 根据1排行列举书籍
func (s *bookService) ListBooks(ctx context.Context, req *dto.ListBooksRequest) (*dto.BookListResponse, error) {
	req.ToStandard()
	books, err := s.bookRepo.ListWithOption(ctx, repository.BookListOption{
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
		bookResponses = append(bookResponses, toBookResponse(s.cfg, book))
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
	err = s.bookRepo.Update(ctx, book)
	if err != nil {
		return nil, err
	}

	return &dto.UploadBookCoverResponse{
		BookID: book.ID,
		Url:    fmt.Sprintf("%s%s", s.cfg.Server.Website, uploadOut.FilePath),
	}, nil
}
