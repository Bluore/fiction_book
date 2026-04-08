package service

import (
	"context"
	"errors"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
)

var (
	ErrUserPurchasedChapterNotFound = errors.New("user purchased chapter not found")
)

type UserPurchasedChapterService interface {
	GetListPurchasedChapterByUserID(ctx context.Context, userID string) (*[]dto.GetUserPurchasedChapterResponse, error)
	GetPurchasedChapterByUserIDAndChapterID(ctx context.Context, userID string, chapterID string) (*dto.GetUserPurchasedChapterResponse, error)
}
type userPurchasedChapterService struct {
	userRepo                 repository.UserRepository
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository
	cfg                      config.Config
}

func NewUserPurchasedChapterService(
	userRepo repository.UserRepository,
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository,
	cfg *config.Config,
) UserPurchasedChapterService {
	return &userPurchasedChapterService{
		userRepo:                 userRepo,
		userPurchasedChapterRepo: userPurchasedChapterRepo,
		cfg:                      *cfg,
	}
}

func (s userPurchasedChapterService) GetListPurchasedChapterByUserID(ctx context.Context, userID string) (*[]dto.GetUserPurchasedChapterResponse, error) {
	purchasedChapters, err := s.userPurchasedChapterRepo.GetListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	chapterRes := make([]dto.GetUserPurchasedChapterResponse, len(*purchasedChapters))
	for i, purchasedChapter := range *purchasedChapters {
		chapterRes[i] = *toUserPurchasedChapterResponse(&purchasedChapter)
	}

	return &chapterRes, nil
}

func (s userPurchasedChapterService) GetPurchasedChapterByUserIDAndChapterID(ctx context.Context, userID string, chapterID string) (*dto.GetUserPurchasedChapterResponse, error) {
	exist, purchasedChapter, err := s.userPurchasedChapterRepo.CheckPurchasedChapter(ctx, userID, chapterID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, ErrUserPurchasedChapterNotFound
	}

	return toUserPurchasedChapterResponse(purchasedChapter), nil
}

func toUserPurchasedChapterResponse(chapter *model.UserPurchasedChapter) *dto.GetUserPurchasedChapterResponse {
	return &dto.GetUserPurchasedChapterResponse{
		PurchasedID: chapter.ID,
		UserID:      chapter.UserID,
		ChapterID:   chapter.ChapterID,
		BookID:      chapter.BookID,
		PurchasedAt: chapter.CreatedAt,
		ActiveAt:    chapter.ActiveAt,
	}
}
