package service

import (
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	"gorm.io/gorm"
)

type OrderService interface {
}

type orderService struct {
	userRepo  repository.UserRepository
	orderRepo repository.OrderRepository
	db        *gorm.DB
	cfg       *config.Config
}

func NewOrderService(
	userRepo repository.UserRepository,
	orderRepo repository.OrderRepository,
	db *gorm.DB,
	cfg *config.Config,
) OrderService {
	return &orderService{
		userRepo:  userRepo,
		orderRepo: orderRepo,
		db:        db,
		cfg:       cfg,
	}
}
