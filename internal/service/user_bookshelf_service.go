package service

import (
	"context"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	"gorm.io/gorm"
)

type UserBookshelfService interface {
	GetUserBookshelfList(ctx context.Context, req *dto.ListUserBookshelfRequest) (*dto.ListUserBookshelfResponse, error)
	CreateUserBookshelf(ctx context.Context, userID string, req *dto.CreateUserBookshelfRequest) error
}

type userBookshelfService struct {
	userRepo          repository.UserRepository
	bookRepo          repository.BookRepository
	userBookshelfRepo repository.UserBookshelfRepository
	db                *gorm.DB
	cfg               *config.Config
}

func NewUserBookshelfService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	db *gorm.DB,
	cfg *config.Config,
) UserBookshelfService {
	return &userBookshelfService{
		userRepo:          userRepo,
		bookRepo:          bookRepo,
		userBookshelfRepo: userBookshelfRepo,
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
	return s.userBookshelfRepo.CreateByUserIDAndBookID(ctx, s.db, userID, req.BookID)
}

func toUserBookshelfResponse(_ *config.Config, in *model.UserBookshelf) *dto.UserBookshelfResponse {
	return &dto.UserBookshelfResponse{
		ID:        in.ID,
		UserID:    in.UserID,
		BookID:    in.BookID,
		CreatedAt: in.CreatedAt,
	}
}
