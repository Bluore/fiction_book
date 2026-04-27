package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	"gorm.io/gorm"
)

type UserBookshelfService interface {
	GetUserBookshelfList(ctx context.Context, req *dto.ListUserBookshelfRequest) (*dto.ListUserBookshelfResponse, error)
	CreateUserBookshelf(ctx context.Context, userID string, req *dto.CreateUserBookshelfRequest) error
	DeleteUserBookshelf(ctx context.Context, userID string, req *dto.DeleteUserBookshelfRequest) error
}

type userBookshelfService struct {
	userRepo          repository.UserRepository
	bookRepo          repository.BookRepository
	userBookshelfRepo repository.UserBookshelfRepository
	bookCatch         cache.BookCache
	limitCatch        cache.LimitCache
	db                *gorm.DB
	cfg               *config.Config
}

func NewUserBookshelfService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	bookCatch cache.BookCache,
	limitCatch cache.LimitCache,
	db *gorm.DB,
	cfg *config.Config,
) UserBookshelfService {
	return &userBookshelfService{
		userRepo:          userRepo,
		bookRepo:          bookRepo,
		userBookshelfRepo: userBookshelfRepo,
		bookCatch:         bookCatch,
		limitCatch:        limitCatch,
		db:                db,
		cfg:               cfg,
	}
}

func (s userBookshelfService) GetUserBookshelfList(ctx context.Context, req *dto.ListUserBookshelfRequest) (*dto.ListUserBookshelfResponse, error) {
	books, err := s.userBookshelfRepo.ListByUserID(ctx, s.db, req.UserID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	bookRes := make([]*dto.UserBookshelfResponse, len(books))
	for i, book := range books {
		bookRes[i] = toUserBookshelfResponse(s.cfg, book)
	}

	count, err := s.userBookshelfRepo.CountByUserID(ctx, s.db, req.UserID)
	if err != nil {
		return nil, err
	}

	page := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: count,
	}

	res := &dto.ListUserBookshelfResponse{
		Books:            bookRes,
		PageListResponse: page,
	}
	return res, nil
}

func (s userBookshelfService) CreateUserBookshelf(ctx context.Context, userID string, req *dto.CreateUserBookshelfRequest) error {
	err := s.db.Transaction(func(db *gorm.DB) error {
		_, err := s.bookRepo.IncreaseOfBookShelf(ctx, db, req.BookID)
		if err != nil {
			return err
		}
		err = s.userBookshelfRepo.CreateByUserIDAndBookID(ctx, s.db, userID, req.BookID)
		if err != nil {
			return err
		}

		exist, _ := s.limitCatch.CheckUserBookshelfBook1Hour(ctx, userID, req.BookID)
		if exist {
			return nil
		}
		// 热力值限流
		_ = s.limitCatch.AddUserBookshelfBook1Hour(ctx, userID, req.BookID)

		_ = s.bookCatch.IncreaseBookThermalDaily(ctx, req.BookID, 10)

		return nil
	})
	if err != nil {
		return err
	}
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
