package service

import (
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
)

type UserPurchasedBookService interface {
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
