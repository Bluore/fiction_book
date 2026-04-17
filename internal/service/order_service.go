package service

import (
	"context"
	"errors"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

const (
	StatusPending = 1  // 待付款
	StatusSending = 2  // 待发货
	StatusSuccess = 3  // 已发货
	StatusDeleted = -1 // 已删除
	StatusOutDate = -2 // 已超时
)

type OrderService interface {
	CreateOrder(ctx context.Context, userId string, req *dto.CreateOrderRequest) (*dto.OrderResponse, error)
	GetOrder(ctx context.Context, userId string, id string) (*dto.OrderResponse, error)
	GetOrderListByUserID(ctx context.Context, userId string, req *dto.GetOrderListRequest) (*dto.GetOrderListResponse, error)
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

func (s orderService) CreateOrder(ctx context.Context, userId string, req *dto.CreateOrderRequest) (*dto.OrderResponse, error) {
	order := model.Order{
		ID:            uuid.New().String(),
		UserId:        userId,
		PayPrice:      req.Price,
		OriginalPrice: req.Price,
		RewordGold:    req.RewardGold,
		Status:        StatusPending,
		PastAt:        time.Now().Add(time.Hour * 24),
		CreatedAt:     time.Now(),
	}

	err := s.db.Transaction(func(db *gorm.DB) error {
		// 货币转化率 1:100
		if int64(req.Price*100) != req.RewardGold {
			return ErrParam
		}

		err := s.orderRepo.CreateOrder(ctx, db, &order)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toOrderResponse(&order), nil
}

func toOrderResponse(in *model.Order) *dto.OrderResponse {
	out := dto.OrderResponse{
		OrderID:       in.ID,
		UserID:        in.UserId,
		PayPrice:      in.PayPrice,
		OriginalPrice: in.OriginalPrice,
		RewordGold:    in.RewordGold,
		PastAt:        in.CreatedAt,
		CreatedAt:     in.CreatedAt,
	}
	if time.Now().Before(in.CreatedAt) {
		out.Status = StatusOutDate
	} else if in.Status == StatusDeleted {
		out.Status = StatusDeleted
	} else {
		out.Status = int(in.Status)
	}
	return &out
}

func (s orderService) GetOrder(ctx context.Context, userId string, id string) (*dto.OrderResponse, error) {
	order, err := s.orderRepo.GetOrderByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.UserId != userId {
		return nil, ErrPermission
	}

	return toOrderResponse(order), nil
}

func (s orderService) GetOrderListByUserID(ctx context.Context, userId string, req *dto.GetOrderListRequest) (*dto.GetOrderListResponse, error) {
	orders, err := s.orderRepo.GetOrderListByUserID(ctx, s.db, userId, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	var outOrder []*dto.OrderResponse
	for _, order := range orders {
		outOrder = append(outOrder, toOrderResponse(order))
	}
	if len(outOrder) == 0 {
		outOrder = make([]*dto.OrderResponse, 0)
	}

	count, err := s.orderRepo.Count(ctx, s.db, userId)
	if err != nil {
		return nil, err
	}

	out := &dto.GetOrderListResponse{
		Orders: outOrder,
		PageListResponse: dto.PageListResponse{
			Page:  req.Page,
			Size:  req.Size,
			Count: count,
		},
	}

	return out, nil
}
