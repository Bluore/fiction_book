package service

import (
	"context"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	"gorm.io/gorm"
)

type UserHistoryService interface {
	GetUserHistoryList(ctx context.Context, req *dto.ListUserHistoryRequest) (*dto.ListUserHistoryResponse, error)
}

type userHistoryService struct {
	userRepo        repository.UserRepository
	bookRepo        repository.BookRepository
	userHistoryRepo repository.UserHistoryRepository
	db              *gorm.DB
	cfg             *config.Config
}

func NewUserHistoryService(
	userRepo repository.UserRepository,
	userHistoryRepo repository.UserHistoryRepository,
	bookRepo repository.BookRepository,
	db *gorm.DB,
	cfg *config.Config,
) UserHistoryService {
	return &userHistoryService{
		userRepo:        userRepo,
		bookRepo:        bookRepo,
		userHistoryRepo: userHistoryRepo,
		db:              db,
		cfg:             cfg,
	}
}

func (s userHistoryService) GetUserHistoryList(ctx context.Context, req *dto.ListUserHistoryRequest) (*dto.ListUserHistoryResponse, error) {
	// 先判断是否存在该用户
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	books, err := s.userHistoryRepo.ListByUserID(ctx, s.db, req.UserID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	bookRes := make([]*dto.UserHistoryResponse, len(books))
	for i, book := range books {
		bookRes[i] = toUserHistoryResponse(s.cfg, book)
	}

	count, err := s.userHistoryRepo.CountByUserID(ctx, s.db, req.UserID)
	if err != nil {
		return nil, err
	}

	page := dto.PageListResponse{
		Page:  req.Page,
		Size:  req.Size,
		Count: count,
	}

	res := &dto.ListUserHistoryResponse{
		Books:            bookRes,
		PageListResponse: page,
	}
	return res, nil
}

func toUserHistoryResponse(_ *config.Config, in *model.UserHistory) *dto.UserHistoryResponse {
	return &dto.UserHistoryResponse{
		ID:        in.ID,
		UserID:    in.UserID,
		BookID:    in.BookID,
		CreatedAt: in.CreatedAt,
	}
}
