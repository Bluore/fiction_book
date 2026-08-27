package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"gorm.io/gorm"
)

type UserBookshelfService interface {
	GetUserBookshelfList(ctx context.Context, userID string, req *dto.PageListRequest) (*dto.BookListResponse, error)
	CreateUserBookshelf(ctx context.Context, userID string, req *dto.CreateUserBookshelfRequest) error
	DeleteUserBookshelf(ctx context.Context, userID string, req *dto.DeleteUserBookshelfRequest) error
}

type userBookshelfService struct {
	userRepo          repository.UserRepository
	bookRepo          repository.BookRepository
	userBookshelfRepo repository.UserBookshelfRepository
	bookCache         cache.BookCache
	bookThermalCache  cache.BookThermalCache
	limitCatch        cache.LimitCache
	db                *gorm.DB
	cfg               *config.Config
	pool              *agilePool.Pool
}

func NewUserBookshelfService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	bookCache cache.BookCache,
	bookThermalCache cache.BookThermalCache,
	limitCatch cache.LimitCache,
	db *gorm.DB,
	cfg *config.Config,
	pool *agilePool.Pool,
) UserBookshelfService {
	return &userBookshelfService{
		userRepo:          userRepo,
		bookRepo:          bookRepo,
		userBookshelfRepo: userBookshelfRepo,
		bookCache:         bookCache,
		bookThermalCache:  bookThermalCache,
		limitCatch:        limitCatch,
		db:                db,
		cfg:               cfg,
		pool:              pool,
	}
}

func (s userBookshelfService) GetUserBookshelfList(ctx context.Context, userID string, req *dto.PageListRequest) (*dto.BookListResponse, error) {
	bookShelfList, err := s.userBookshelfRepo.ListByUserID(ctx, s.db, userID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	var bookRes []*dto.BookResponse
	for _, bookShelfItem := range bookShelfList {
		bookCache, bookCacheHit, _ := s.bookCache.GetBook(ctx, bookShelfItem.BookID)
		if !bookCacheHit {
			var err error
			bookCache, err = s.bookRepo.GetByID(ctx, bookShelfItem.BookID)
			if err != nil {
				return nil, err
			}

			_ = s.bookCache.UpdateBook(ctx, bookCache.ID, bookCache)
		}

		bookRes = append(bookRes, toBookResponse(s.cfg, bookCache))
	}

	count, err := s.userBookshelfRepo.CountByUserID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}

	if len(bookRes) == 0 {
		bookRes = make([]*dto.BookResponse, 0)
	}

	page := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: count,
	}

	res := &dto.BookListResponse{
		Books:            bookRes,
		PageListResponse: page,
	}
	return res, nil
}

func (s userBookshelfService) CreateUserBookshelf(ctx context.Context, userID string, req *dto.CreateUserBookshelfRequest) error {
	var err error
	err = s.userBookshelfRepo.CreateByUserIDAndBookID(ctx, s.db, userID, req.BookID)
	if err != nil {
		return err
	}

	s.pool.SubmitCtx(ctx, agilePool.TaskFunc(func() error {
		return bookThermalCacheAddByBookshelfTask(ctx, &s, userID, req)
	}))

	return nil
}

func toUserBookshelfResponse(_ *config.Config, in *model.UserBookshelf) *dto.UserBookshelfResponse {
	return &dto.UserBookshelfResponse{
		ID:        in.ID,
		UserID:    in.UserID,
		BookID:    in.BookID,
		CreatedAt: in.CreatedAt,
	}
}

func (s userBookshelfService) DeleteUserBookshelf(ctx context.Context, userID string, req *dto.DeleteUserBookshelfRequest) error {
	err := s.db.Transaction(func(db *gorm.DB) error {
		exist, _, err := s.userBookshelfRepo.CheckByUserIDAndBookID(ctx, db, userID, req.BookID)
		if err != nil {
			return err
		}
		if !exist {
			return ErrBookNotFound
		}

		err = s.userBookshelfRepo.DeleteByUserIDAndBookID(ctx, db, userID, req.BookID)
		return err
	})
	if err != nil {
		return err
	}
	return nil
}
