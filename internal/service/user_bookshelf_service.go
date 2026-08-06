package service

import (
	"context"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"go.uber.org/zap"
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

			_ = s.bookCache.UpdateBook(ctx, bookCache)
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

func bookThermalCacheAddByBookshelfTask(ctx context.Context, s *userBookshelfService, userID string, req *dto.CreateUserBookshelfRequest) error {
	exist, _ := s.limitCatch.CheckUserBookshelfBook1Hour(ctx, userID, req.BookID)
	if exist {
		return nil
	}

	// 热力值限流
	_ = s.limitCatch.AddUserBookshelfBook1Hour(ctx, userID, req.BookID)

	// 更新数据
	s.pool.SubmitCtx(constant.DoneServerCtx, agilePool.TaskFunc(func() error {
		return bookThermalCacheAddValueTask(constant.DoneServerCtx, s.db, s.bookThermalCache, s.bookCache, s.bookRepo, s.pool, userID, 10)
	}))

	return nil
}

func bookThermalCacheAddValueTask(ctx context.Context, db *gorm.DB, bookThermalCache cache.BookThermalCache, bookCache cache.BookCache, d repository.BookRepository, p *agilePool.Pool, bookID string, value cache.BookThermalValue) error {
	err := bookThermalCache.AddBookThermalValue(ctx, bookID, value)
	updateBookThermalValueFunc := func() error {
		bookDetail, err := d.GetByID(ctx, bookID)
		if err != nil {
			return err
		}
		if bookDetail == nil {
			return ErrBookNotFound
		}
		bookThermalValue := cache.BookThermalValue(bookDetail.ThermalValue)
		bookThermalValue += value

		err = bookThermalCache.UpdateBookThermalValue(ctx, bookID, bookThermalValue)
		if err != nil {
			logger.Error("UpdateBookThermalValue", zap.Error(err))
			return err
		}
		return nil
	}
	if err != nil {
		if errors.Is(err, cache.ErrorBookThermalValueNotExist) {
			err := updateBookThermalValueFunc()
			if err != nil {
				return err
			}

			goto breakBookCacheError
		}
		return err
	}
breakBookCacheError:

	if err := bookThermalCache.LockBookThermalValue(ctx, bookID); err != nil {
		if errors.Is(err, cache.ErrorBookThermalValueNotExist) {
			return nil
		}
		return err
	}

	// 异步：数据库同步
	p.SubmitCtx(ctx, agilePool.TaskFunc(func() error {
		return bookThermalCacheSyncToDatabaseTask(ctx, db, bookThermalCache, bookCache, d, bookID)
	}))

	// 更新榜单
	err = bookCache.IncreaseBookThermalDaily(ctx, bookID, int(value))
	if err != nil {
		logger.Error("IncreaseBookThermalDaily", zap.Error(err))
	}

	return nil
}

func bookThermalCacheSyncToDatabaseTask(ctx context.Context, db *gorm.DB, bookThermalCache cache.BookThermalCache, bookCache cache.BookCache, d repository.BookRepository, bookID string) error {
	bookThermalValue, ok, _ := bookThermalCache.GetBookThermalValue(ctx, bookID)
	if !ok {
		return nil
	}

	err := d.UpdateBookThermal(ctx, db, bookID, bookThermalValue)
	if err != nil {
		return err
	}

	bookDetail, err := d.GetByID(ctx, bookID)
	if err != nil {
		return err
	}
	err = bookCache.UpdateBook(ctx, bookDetail)
	if err != nil {
		logger.Error("UpdateBookCache", zap.Error(err))
	}

	return nil
}
