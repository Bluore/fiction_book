package service

import (
	"context"

	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/response"
)

type UserPurchasedBookService interface {
	GetListPurchasedBookByUserID(ctx context.Context, userID string) (*[]dto.GetUserPurchasedBookResponse, error)
	GetPurchasedBookByUserIDANDBookID(ctx context.Context, userID string, bookID string) (*dto.GetUserPurchasedBookResponse, error)
}
type userPurchasedBookService struct {
	userRepo              repository.UserRepository
	userPurchasedBookPepo repository.UserPurchasedBookRepository
	cfg                   config.Config
}

func NewUserPurchasedBookService(
	userRepo repository.UserRepository,
	userPurchasedBookPepo repository.UserPurchasedBookRepository,
	cfg *config.Config,
) UserPurchasedBookService {
	return &userPurchasedBookService{
		userRepo:              userRepo,
		userPurchasedBookPepo: userPurchasedBookPepo,
		cfg:                   *cfg,
	}
}

func (s userPurchasedBookService) GetListPurchasedBookByUserID(ctx context.Context, userID string) (*[]dto.GetUserPurchasedBookResponse, error) {
	purchasedBooks, err := s.userPurchasedBookPepo.GetListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	bookRes := make([]dto.GetUserPurchasedBookResponse, len(*purchasedBooks))
	for i, purchasedBook := range *purchasedBooks {
		bookRes[i] = *toUserPurchasedBookResponse(&purchasedBook)
	}

	return &bookRes, nil
}

func (s userPurchasedBookService) GetPurchasedBookByUserIDANDBookID(ctx context.Context, userID string, bookID string) (*dto.GetUserPurchasedBookResponse, error) {
	exist, purchasedBook, err := s.userPurchasedBookPepo.CheckPurchasedBook(ctx, userID, bookID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, response.ErrUserPurchasedBookNotFound
	}

	return toUserPurchasedBookResponse(purchasedBook), nil
}

func toUserPurchasedBookResponse(book *model.UserPurchasedBook) *dto.GetUserPurchasedBookResponse {
	return &dto.GetUserPurchasedBookResponse{
		PurchasedID: book.ID,
		UserID:      book.UserID,
		BookID:      book.BookID,
		PurchasedAt: book.CreatedAt,
		ActiveAt:    book.ActiveAt,
	}
}
